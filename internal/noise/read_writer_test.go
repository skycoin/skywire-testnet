package noise

import (
	"net"
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
