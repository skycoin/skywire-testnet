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

func Example_transport_InMemoryPubKeyTable() {
	/*
	   This test requires IPs on host
	   sudo ip addr add 192.168.1.2 dev  lo
	   sudo ip addr add 192.168.1.3 dev  lo
	*/

	if true {
		return
	}

	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()

	addr1, err := net.ResolveTCPAddr("tcp", "192.168.1.2:9119")
	l1, err := net.ListenTCP("tcp", addr1)

	addr2, err := net.ResolveTCPAddr("tcp", "192.168.1.3:9119")
	l2, err := net.ListenTCP("tcp", addr2)

	entries := map[cipher.PubKey]string{
		pk2: addr2.String(),
		pk1: addr1.String(),
	}

	pkt1 := transport.InMemoryPubKeyTable(entries)
	pkt2 := transport.InMemoryPubKeyTable(entries)

	f1 := &transport.TCPFactory{pk1, pkt1, l1}
	f2 := &transport.TCPFactory{pk2, pkt2, l2}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		tr, err := f1.Accept(context.TODO())
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

	tr, err := f2.Dial(context.TODO(), pk1)
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
	wg.Wait()
	fmt.Println("Finish")

	// Output: Dial success: true
	// Accept success: true
	// Write success: true
	// Message recieved: Hallo!
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

	pkt1 := transport.InMemoryPubKeyTable(map[cipher.PubKey]string{pk2: addr2.String()})
	pkt2 := transport.InMemoryPubKeyTable(map[cipher.PubKey]string{pk1: addr1.String()})

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

func Example_transport_FilePubKetTable() {

	pkLine := func(seed, addr string) string {
		pk, _, _ := cipher.GenerateDeterministicKeyPair([]byte(seed))
		return fmt.Sprintf("%s\t%s\n", pk, addr)
	}

	pkFileContent :=
		fmt.Sprintf("%v%v", pkLine("tcp-tr nodeA", "192.168.1.2:9119"),
			pkLine("tcp-tr nodeB", "192.168.1.3:9119"))
	fmt.Printf("pubkeys:\n%v", pkFileContent)

	tmpfile, _ := ioutil.TempFile("", "pktable")
	defer os.Remove(tmpfile.Name())

	_, _ = tmpfile.Write([]byte(pkFileContent))

	pkt, err := transport.FilePubKeyTable(tmpfile.Name())
	fmt.Printf("Opening FilePubKeyTable success: %v\n", err == nil)

	fmt.Printf("PK for 192.168.1.2:9119: %v\n", pkt.RemotePK("192.168.1.2:9119"))

	// Output: pubkeys:
	// 0322f66c5cac131376a2e6a20c6e31511baa7ec7bb0dccd26954c86c94af7d02b7	192.168.1.2:9119
	// 021adbbdf76f223f6e25ffe0b5626e600a1fcbb5fbbda833147a262a61b21312f3	192.168.1.3:9119
	// Opening FilePubKeyTable success: true
	// PK for 192.168.1.2:9119 0322f66c5cac131376a2e6a20c6e31511baa7ec7bb0dccd26954c86c94af7d02b7
}
