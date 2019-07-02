package noise

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReadWriter(t *testing.T) {

	type Result struct {
		n   int
		err error
		b   []byte
	}

	t.Run("concurrent", func(t *testing.T) {
		aPK, aSK := cipher.GenerateKeyPair()
		bPK, bSK := cipher.GenerateKeyPair()

		aNs, err := KKAndSecp256k1(Config{
			LocalPK:   aPK,
			LocalSK:   aSK,
			RemotePK:  bPK,
			Initiator: true,
		})
		require.NoError(t, err)

		bNs, err := KKAndSecp256k1(Config{
			LocalPK:   bPK,
			LocalSK:   bSK,
			RemotePK:  aPK,
			Initiator: false,
		})
		require.NoError(t, err)

		aConn, bConn := net.Pipe()
		defer func() {
			_ = aConn.Close() //nolint:errcheck
			_ = bConn.Close() //nolint:errcheck
		}()

		aRW := NewReadWriter(aConn, aNs)
		bRW := NewReadWriter(bConn, bNs)

		hsCh := make(chan error, 2)
		defer close(hsCh)
		go func() { hsCh <- aRW.Handshake(time.Second) }()
		go func() { hsCh <- bRW.Handshake(time.Second) }()
		require.NoError(t, <-hsCh)
		require.NoError(t, <-hsCh)

		const groupSize = 10
		const totalGroups = 5
		const msgCount = totalGroups * groupSize

		writes := make([][]byte, msgCount)

		wCh := make(chan Result, msgCount)
		defer close(wCh)
		rCh := make(chan Result, msgCount)
		defer close(rCh)

		for i := 0; i < msgCount; i++ {
			writes[i] = []byte(fmt.Sprintf("this is message: %d", i))
		}

		for i := 0; i < totalGroups; i++ {
			go func(i int) {
				for j := 0; j < groupSize; j++ {
					go func(i, j int) {
						b := writes[i*j]
						n, err := aRW.Write(b)
						wCh <- Result{n: n, err: err, b: b}
					}(i, j)
					go func() {
						buf := make([]byte, 100)
						n, err := bRW.Read(buf)
						rCh <- Result{n: n, err: err, b: buf[:n]}
					}()
				}
			}(i)
		}

		for i := 0; i < msgCount; i++ {
			w := <-wCh
			fmt.Printf("write_result[%d]: b(%s) err(%v)\n", i, string(w.b), w.err)
			assert.NoError(t, w.err)
			assert.True(t, w.n > 0)

			r := <-rCh
			fmt.Printf(" read_result[%d]: b(%s) err(%v)\n", i, string(r.b), r.err)
			assert.NoError(t, r.err)
			assert.True(t, r.n > 0)
		}
	})
}

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
