package noise

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestReadWriterKKPattern(t *testing.T) {
	pkI, skI := cipher.GenerateKeyPair()
	pkR, skR := cipher.GenerateKeyPair()

	confI := Config{
		LocalPK:   pkI,
		LocalSK:   skI,
		RemotePK:  pkR,
		Initiator: true,
	}

	confR := Config{
		LocalPK:   pkR,
		LocalSK:   skR,
		RemotePK:  pkI,
		Initiator: false,
	}

	nI, err := KKAndSecp256k1(confI)
	require.NoError(t, err)

	nR, err := KKAndSecp256k1(confR)
	require.NoError(t, err)

	connI, connR := net.Pipe()
	rwI := NewReadWriter(connI, nI)
	rwR := NewReadWriter(connR, nR)

	errCh := make(chan error)
	go func() { errCh <- rwR.Handshake(time.Second) }()
	require.NoError(t, rwI.Handshake(time.Second))
	require.NoError(t, <-errCh)

	go func() {
		_, err := rwI.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 3)
	n, err := rwR.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	go func() {
		_, err := rwI.Read(buf)
		errCh <- err
	}()

	n, err = rwR.Write([]byte("bar"))
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)
}

func TestReadWriterXKPattern(t *testing.T) {
	pkI, skI := cipher.GenerateKeyPair()
	pkR, skR := cipher.GenerateKeyPair()

	confI := Config{
		LocalPK:   pkI,
		LocalSK:   skI,
		RemotePK:  pkR,
		Initiator: true,
	}

	confR := Config{
		LocalPK:   pkR,
		LocalSK:   skR,
		Initiator: false,
	}

	nI, err := XKAndSecp256k1(confI)
	require.NoError(t, err)

	nR, err := XKAndSecp256k1(confR)
	require.NoError(t, err)

	connI, connR := net.Pipe()
	rwI := NewReadWriter(connI, nI)
	rwR := NewReadWriter(connR, nR)

	errCh := make(chan error)
	go func() { errCh <- rwR.Handshake(time.Second) }()
	require.NoError(t, rwI.Handshake(time.Second))
	require.NoError(t, <-errCh)

	go func() {
		_, err := rwI.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 3)
	n, err := rwR.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("foo"), buf)

	go func() {
		_, err := rwI.Read(buf)
		errCh <- err
	}()

	n, err = rwR.Write([]byte("bar"))
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("bar"), buf)
}

func TestReadWriterConcurrentTCP(t *testing.T) {
	const readCount = 15
	readErrs := make([]error, readCount)
	writeErrs := make([]error, readCount)
	msg := []byte("foo")

	errNoOp := errors.New("no operation")
	for i := 0; i < readCount; i++ {
		readErrs[i] = errNoOp
		writeErrs[i] = errNoOp
	}

	l, err := net.Listen("tcp", ":0") // nolint: gosec
	require.NoError(t, err)
	defer l.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}

		var rwg sync.WaitGroup
		for i := 0; i < readCount; i++ {
			rwg.Add(1)
			go func(idx int, c net.Conn) {
				buf := make([]byte, 3)
				if _, err := c.Read(buf); err != nil {
					readErrs[idx] = err
				}

				if string(buf) != "foo" {
					readErrs[idx] = errors.New("invalid message")
				}

				readErrs[idx] = nil
				rwg.Done()
			}(i, conn)
		}
		rwg.Wait()
		wg.Done()
	}()

	conn, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	for i := 0; i < readCount; i++ {
		wg.Add(1)
		go func(idx int, c net.Conn) {
			if _, err := c.Write(msg); err != nil {
				writeErrs[idx] = err
			}

			writeErrs[idx] = nil
			wg.Done()
		}(i, conn)
	}
	wg.Wait()

	for i := 0; i < readCount; i++ {
		require.NoError(t, readErrs[i], fmt.Sprintf("read #%d", i))
		require.NoError(t, writeErrs[i], fmt.Sprintf("write #%d", i))
	}
}
