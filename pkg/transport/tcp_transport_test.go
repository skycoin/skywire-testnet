package transport_test

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

	"github.com/skycoin/skywire/pkg/transport"
)

/*
   This test requires IPs on host
   sudo ip addr add 192.168.1.2 dev  lo
   sudo ip addr add 192.168.1.3 dev  lo
*/

func Example_transport_TCPFactory() {
	pkA := pkFromSeed("nodeA")
	pkB := pkFromSeed("nodeB")
	ipA := "192.168.1.2:9119"
	ipB := "192.168.1.3:9119"

	addrA, _ := net.ResolveTCPAddr("tcp", ipA)
	lsnA, err := net.ListenTCP("tcp", addrA)
	if err != nil {
		fmt.Println(err)
	}

	addrB, _ := net.ResolveTCPAddr("tcp", ipB)
	lsnB, err := net.ListenTCP("tcp", addrB)
	if err != nil {
		fmt.Println(err)
	}

	pkt := transport.MemoryPubKeyTable(
		map[cipher.PubKey]string{
			pkA: addrA.String(),
			pkB: addrB.String(),
		})

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()

		fA := &transport.TCPFactory{pkA, pkt, lsnA}
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
		fB := &transport.TCPFactory{pkB, pkt, lsnB}
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

	pkt1 := transport.MemoryPubKeyTable(map[cipher.PubKey]string{pk2: addr2.String()})
	pkt2 := transport.MemoryPubKeyTable(map[cipher.PubKey]string{pk1: addr1.String()})

	fmt.Println(addr1.String())
	fmt.Println(addr2.String())

	f1 := &transport.TCPFactory{pk1, pkt1, l1}
	assert.Equal(t, "tcp", f1.Type())
	assert.Equal(t, pk1, f1.Local())

	f2 := &transport.TCPFactory{pk2, pkt2, l2}
	assert.Equal(t, "tcp", f2.Type())
	assert.Equal(t, pk2, f2.Local())

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

	require.NoError(t, <-errAcceptCh)
	require.NoError(t, <-errDialCh)
	require.NoError(t, f2.Close())
	require.NoError(t, f1.Close())
}

func TestFilePKTable(t *testing.T) {
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

	pkt, err := transport.FilePubKeyTable(tmpfile.Name())
	require.NoError(t, err)

	raddr := pkt.RemoteAddr(pk)
	assert.Equal(t, addr.String(), raddr)

	rpk := pkt.RemotePK(addr.String())
	assert.Equal(t, pk, rpk)
}

func pkFromSeed(seed string) cipher.PubKey {
	pk, _, err := cipher.GenerateDeterministicKeyPair([]byte(seed))
	if err != nil {
		return cipher.PubKey{}
	}
	return pk
}

func Example_transport_MemoryPubKeyTable() {
	pkA, pkB := pkFromSeed("nodeA"), pkFromSeed("nodeB")
	ipA, ipB := "192.168.1.2:9119", "192.168.1.3:9119"
	ipAA := "192.168.1.2:54312"
	entries := map[cipher.PubKey]string{
		pkA: ipA,
		pkB: ipB,
	}
	pkt := transport.MemoryPubKeyTable(entries)

	fmt.Printf("ipA: %v\n", pkt.RemoteAddr(pkA))
	fmt.Printf("pkB in: %v\n", pkt.RemotePK(ipA))
	fmt.Printf("pkA out: %v\n", pkt.RemotePK(ipAA))

	// Output: ipA: 192.168.1.2:9119
	// pkB in: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab
	// pkA out: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab
}

func Example_transport_FilePubKeyTable() {
	pkA, pkB := pkFromSeed("nodeA"), pkFromSeed("nodeB")
	ipA, ipB := "192.168.1.2:9119", "192.168.1.3:9119"
	ipAA := "192.168.1.2:54312"

	pkFileContent :=
		fmt.Sprintf("%v%v",
			fmt.Sprintf("%s\t%s\n", pkA, ipA),
			fmt.Sprintf("%s\t%s\n", pkB, ipB))
	fmt.Printf("pubkeys:\n%v", pkFileContent)

	tmpfile, _ := ioutil.TempFile("", "pktable")

	_, err := tmpfile.Write([]byte(pkFileContent))
	fmt.Printf("Write file success: %v\n", err == nil)

	pkt, err := transport.FilePubKeyTable(tmpfile.Name())
	// pkt.RemoteAddr(pkFromSeed("nodeA"))
	// pkt.RemoteAddr()

	fmt.Printf("Opening FilePubKeyTable success: %v\n", err == nil)

	fmt.Printf("ipA: %v\n", pkt.RemoteAddr(pkA))
	fmt.Printf("PK for ipA: %v\n", pkt.RemotePK(ipA))
	fmt.Printf("PK for ipAA: %v\n", pkt.RemotePK(ipAA))
	fmt.Printf("PK for ipB: %v\n", pkt.RemotePK(ipB))

	// Output: pubkeys:
	// 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab	192.168.1.2:9119
	// 033978326862c191eaa39e33bb556a6296466facfe36bfb81e6b4c99d9c510e09f	192.168.1.3:9119
	// Write file success: true
	// Opening FilePubKeyTable success: true
	// ipA: 192.168.1.2:9119
	// PK for ipA: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab
	// PK for ipAA: 03c8ab0302ecda8564df4bce595c456a03b64871caff699fcafaf24a93058474ab
	// PK for ipB: 033978326862c191eaa39e33bb556a6296466facfe36bfb81e6b4c99d9c510e09f
}
