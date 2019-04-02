package transport

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

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

	pkt1 := InMemoryPubKeyTable(map[cipher.PubKey]*net.TCPAddr{pk2: addr2})
	pkt2 := InMemoryPubKeyTable(map[cipher.PubKey]*net.TCPAddr{pk1: addr1})

	f1 := NewTCPFactory(pk1, pkt1, l1)
	errCh := make(chan error)
	go func() {
		tr, err := f1.Accept(context.TODO())
		if err != nil {
			errCh <- err
			return
		}

		if _, err := tr.Write([]byte("foo")); err != nil {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	f2 := NewTCPFactory(pk2, pkt2, l2)
	assert.Equal(t, "tcp", f2.Type())
	assert.Equal(t, pk2, f2.Local())

	tr, err := f2.Dial(context.TODO(), pk1)
	require.NoError(t, err)
	assert.Equal(t, "tcp", tr.Type())
	// assert.Equal(t, pk2, tr.Local())
	// assert.Equal(t, pk1, tr.Remote())

	buf := make([]byte, 3)
	_, err = tr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), buf)

	require.NoError(t, tr.Close())
	require.NoError(t, f2.Close())
	require.NoError(t, f1.Close())
}

func TestFilePKTable(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()

	tmpfile, err := ioutil.TempFile("", "pktable")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9000")
	require.NoError(t, err)

	_, err = tmpfile.Write([]byte(fmt.Sprintf("%s\t%s\n", pk, addr)))
	require.NoError(t, err)

	pkt, err := FilePubKeyTable(tmpfile.Name())
	require.NoError(t, err)

	raddr := pkt.RemoteAddr(pk)
	assert.Equal(t, addr, raddr)

	rpk := pkt.RemotePK(addr.IP)
	assert.Equal(t, pk, rpk)
}
