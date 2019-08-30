// +build !no_ci

package snet_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	th "github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/snet"
)

/*
   TCP-Transport tests requires preconfigured IP aliases on on host

   Linux: `for ((i=1; i<=255; i++)){sudo ip addr add 12.12.12.$i/32 dev lo}`

   MacOS:
   ```bash
   $ brew install iproute2mac
   $ for ((i=1; i<=255; i++)){sudo ip addr add 12.12.12.$i/32 dev lo0} # note lo0
   ```
*/

func pkFromSeed(seed string) cipher.PubKey {
	pk, _, err := cipher.GenerateDeterministicKeyPair([]byte(seed))
	if err != nil {
		return cipher.PubKey{}
	}
	return pk
}

func Example_transport_TCPFactory() {
	pkA := pkFromSeed("12.12.12.1")
	pkB := pkFromSeed("12.12.12.2")
	ipA := "12.12.12.1:9119"
	ipB := "12.12.12.2:9119"

	addrA, errA := net.ResolveTCPAddr("tcp", ipA)
	if errA != nil {
		fmt.Printf("net.ResolveTCPAddr(\"tcp\", ipA) failed: %v\n", errA)
	}

	lsnA, err := net.ListenTCP("tcp", addrA)
	if err != nil {
		fmt.Println(err)
	}

	addrB, errB := net.ResolveTCPAddr("tcp", ipB)
	if errB != nil {
		fmt.Printf("net.ResolveTCPAddr(\"tcp\", ipA) failed: %v\n", errB)
	}

	lsnB, err := net.ListenTCP("tcp", addrB)
	if err != nil {
		fmt.Println(err)
	}

	pkt := snet.MakeMemoryPubKeyTable(
		map[cipher.PubKey]string{
			pkA: addrA.String(),
			pkB: addrB.String(),
		})

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()

		fA := snet.NewTCPFactory(pkA, pkt, lsnA)
		tr, err := fA.Accept(context.TODO())
		if err != nil {
			fmt.Printf("Accept err: %v\n", err)
			return
		}
		fmt.Printf("Accept success: %v\n", err == nil)

		if _, err := tr.Write([]byte("Hallo!")); err != nil {
			fmt.Printf("Write err: %v\n", err)
			return
		}
		fmt.Printf("Write success: %v\n", err == nil)
		return
	}()

	go func() {
		defer wg.Done()
		fB := snet.NewTCPFactory(pkB, pkt, lsnB)
		tr, err := fB.Dial(context.TODO(), pkA)
		if err != nil {
			fmt.Printf("Dial err: %v\n", err)
		}
		fmt.Printf("Dial success: %v\n", err == nil)

		buf := make([]byte, 6)
		_, err = tr.Read(buf)
		if err != nil {
			fmt.Printf("Read err: %v\n", err)
		}

		fmt.Printf("Message recieved: %s\n", buf)
	}()
	wg.Wait()

	fmt.Println("Finish")

	// Unordered output: Accept success: true
	// Write success: true
	// Dial success: true
	// Message recieved: Hallo!
	// Finish
}

func TestTCPFactory(t *testing.T) {
	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()

	addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9000")
	require.NoError(t, err)
	l1, err := net.ListenTCP("tcp", addr1)
	require.NoError(t, err)

	addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9001")
	require.NoError(t, err)
	l2, err := net.ListenTCP("tcp", addr2)
	require.NoError(t, err)

	pkt1 := snet.MakeMemoryPubKeyTable(map[cipher.PubKey]string{pk2: addr2.String()})
	pkt2 := snet.MakeMemoryPubKeyTable(map[cipher.PubKey]string{pk1: addr1.String()})

	f1 := snet.NewTCPFactory(pk1, pkt1, l1)
	f2 := snet.NewTCPFactory(pk2, pkt2, l2)
	require.Equal(t, "tcp", f1.Type())
	require.Equal(t, pk1, f1.Local())
	require.Equal(t, "tcp", f2.Type())
	require.Equal(t, pk2, f2.Local())

	var wg sync.WaitGroup

	wg.Add(2)
	errAcceptCh := make(chan error)
	go func() {
		tr, err := f1.Accept(context.TODO())
		if err != nil {
			errAcceptCh <- err
			return
		}

		if _, err := tr.Write([]byte("Hello!")); err != nil {
			errAcceptCh <- err
			return
		}

		require.NoError(t, tr.Close())
		close(errAcceptCh)
		wg.Done()
	}()

	errDialCh := make(chan error)
	go func() {
		tr, err := f2.Dial(context.TODO(), pk1)
		if err != nil {
			errDialCh <- err
		}

		buf := make([]byte, 6)
		_, err = tr.Read(buf)
		if err != nil {
			errDialCh <- err
		}

		assert.Equal(t, []byte("Hello!"), buf)
		require.NoError(t, tr.Close())

		close(errDialCh)
		wg.Done()
	}()

	wg.Wait()

	th.NoErrorN(t, <-errAcceptCh, <-errDialCh,
		f2.Close(), f1.Close())

}

func TestFilePubKeyTable(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	tmpfile, err := ioutil.TempFile("", "pktable")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Remove(tmpfile.Name()))
	}()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9000")
	require.NoError(t, err)

	_, err = tmpfile.Write([]byte(fmt.Sprintf("%s\t%s\n", pk, addr)))
	require.NoError(t, err)

	pkt, err := snet.FilePubKeyTable(tmpfile.Name())
	require.NoError(t, err)
	require.Equal(t, pkt.Count(), 1)

	raddr := pkt.RemoteAddr(pk)
	assert.Equal(t, addr.IP.String(), raddr.IP.String())

	rpk, ok := pkt.RemotePK(addr.String())
	assert.True(t, ok)
	assert.Equal(t, pk, rpk)
}

func Example_transport_MemoryPubKeyTable() {
	pkA, pkB := pkFromSeed("nodeA"), pkFromSeed("nodeB")
	ipA, ipB := "12.12.12.1:9119", "skyhost_003:9119"
	ipAA := "12.12.12.1:54312"
	entries := map[cipher.PubKey]string{
		pkA: ipA,
		pkB: ipB,
	}
	pkt := snet.MakeMemoryPubKeyTable(entries)

	fmt.Printf("ipA: %v\n", pkt.RemoteAddr(pkA))
	rpkA, errA := pkt.RemotePK(ipA)
	fmt.Printf("pkB in: %v ok: %v\n", rpkA, errA)
	rpkAA, errAA := pkt.RemotePK(ipAA)
	fmt.Printf("pkA out: %v ok: %v\n", rpkAA, errAA)

	// Output: ipA: 12.12.12.1:9119
	// pkB in: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab ok: true
	// pkA out: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab ok: true
}

func Example_transport_FilePubKeyTable() {
	pkA, pkB := pkFromSeed("nodeA"), pkFromSeed("nodeB")
	ipA, ipB := "12.12.12.1:9119", "12.12.12.2:9119"
	ipAA := "12.12.12.1:54312"

	pkFileContent :=
		fmt.Sprintf("%v%v",
			fmt.Sprintf("%s\t%s\n", pkA, ipA),
			fmt.Sprintf("%s\t%s\n", pkB, ipB))
	fmt.Printf("pubkeys:\n%v", pkFileContent)

	tmpfile, _ := ioutil.TempFile("", "pktable")

	_, err := tmpfile.Write([]byte(pkFileContent))
	fmt.Printf("Write file success: %v\n", err == nil)

	pkt, err := snet.FilePubKeyTable(tmpfile.Name())

	fmt.Printf("Opening FilePubKeyTable success: %v\n", err == nil)
	fmt.Printf("ip for %v: %v\n", pkA, pkt.RemoteAddr(pkA))
	for _, ip := range []string{ipA, ipAA, ipB} {
		pk, ok := pkt.RemotePK(ip)
		fmt.Printf("PK for %s: %v ok: %v\n", ip, pk, ok)
	}

	// Output: pubkeys:
	// 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab	12.12.12.1:9119
	// 033978326862c191eaa39e33bb556a6296466facfe36bfb81e6b4c99d9c510e09f	12.12.12.2:9119
	// Write file success: true
	// Opening FilePubKeyTable success: true
	// ip for 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab: 12.12.12.1:9119
	// PK for 12.12.12.1:9119: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab ok: true
	// PK for 12.12.12.1:54312: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab ok: true
	// PK for 12.12.12.2:9119: 033978326862c191eaa39e33bb556a6296466facfe36bfb81e6b4c99d9c510e09f ok: true
}
