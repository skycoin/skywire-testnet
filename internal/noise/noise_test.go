package noise

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestKKAndSecp256k1(t *testing.T) {
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

	// -> e, es
	msg, err := nI.HandshakeMessage()
	require.NoError(t, err)
	require.Error(t, nR.ProcessMessage(append(msg, 1)))
	require.NoError(t, nR.ProcessMessage(msg))

	// <- e, ee
	msg, err = nR.HandshakeMessage()
	require.NoError(t, err)
	require.Error(t, nI.ProcessMessage(append(msg, 1)))
	require.NoError(t, nI.ProcessMessage(msg))

	require.True(t, nI.HandshakeFinished())
	require.True(t, nR.HandshakeFinished())

	encrypted := nI.EncryptUnsafe([]byte("foo"))
	decrypted, err := nR.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), decrypted)

	encrypted = nR.EncryptUnsafe([]byte("bar"))
	decrypted, err = nI.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("bar"), decrypted)

	encrypted = nI.EncryptUnsafe([]byte("baz"))
	decrypted, err = nR.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("baz"), decrypted)
}

func TestXKAndSecp256k1(t *testing.T) {
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

	// -> e, es
	msg, err := nI.HandshakeMessage()
	require.NoError(t, err)
	require.NoError(t, nR.ProcessMessage(msg))

	// <- e, ee
	msg, err = nR.HandshakeMessage()
	require.NoError(t, err)
	require.NoError(t, nI.ProcessMessage(msg))

	// -> s, se
	msg, err = nI.HandshakeMessage()
	require.NoError(t, err)
	require.NoError(t, nR.ProcessMessage(msg))

	require.True(t, nI.HandshakeFinished())
	require.True(t, nR.HandshakeFinished())

	encrypted := nI.EncryptUnsafe([]byte("foo"))
	decrypted, err := nR.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("foo"), decrypted)

	encrypted = nR.EncryptUnsafe([]byte("bar"))
	decrypted, err = nI.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("bar"), decrypted)

	encrypted = nI.EncryptUnsafe([]byte("baz"))
	decrypted, err = nR.DecryptUnsafe(encrypted)
	require.NoError(t, err)
	assert.Equal(t, []byte("baz"), decrypted)
}
