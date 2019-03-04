package cipher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubKeyString(t *testing.T) {
	p, _ := GenerateKeyPair()
	require.Equal(t, p.Hex(), p.String())
}

func TestPubKeyTextMarshaller(t *testing.T) {
	p, _ := GenerateKeyPair()
	h, err := p.MarshalText()
	require.NoError(t, err)

	var p2 PubKey
	err = p2.UnmarshalText(h)
	require.NoError(t, err)
	require.Equal(t, p, p2)
}

func TestPubKeyBinaryMarshaller(t *testing.T) {
	p, _ := GenerateKeyPair()
	b, err := p.MarshalBinary()
	require.NoError(t, err)

	var p2 PubKey
	err = p2.UnmarshalBinary(b)
	require.NoError(t, err)
	require.Equal(t, p, p2)
}

func TestSecKeyString(t *testing.T) {
	_, s := GenerateKeyPair()
	require.Equal(t, s.Hex(), s.String())
}

func TestSecKeyTextMarshaller(t *testing.T) {
	_, s := GenerateKeyPair()
	h, err := s.MarshalText()
	require.NoError(t, err)

	var s2 SecKey
	err = s2.UnmarshalText(h)
	require.NoError(t, err)
	require.Equal(t, s, s2)
}

func TestSecKeyBinaryMarshaller(t *testing.T) {
	_, s := GenerateKeyPair()
	b, err := s.MarshalBinary()
	require.NoError(t, err)

	var s2 SecKey
	err = s2.UnmarshalBinary(b)
	require.NoError(t, err)
	require.Equal(t, s, s2)
}

func TestSigString(t *testing.T) {
	_, sk := GenerateKeyPair()
	sig, err := SignPayload([]byte("foo"), sk)
	require.NoError(t, err)
	assert.Equal(t, sig.Hex(), sig.String())
}

func TestSigTextMarshaller(t *testing.T) {
	_, sk := GenerateKeyPair()
	sig, err := SignPayload([]byte("foo"), sk)
	require.NoError(t, err)
	h, err := sig.MarshalText()
	require.NoError(t, err)

	var sig2 Sig
	err = sig2.UnmarshalText(h)
	require.NoError(t, err)
	assert.Equal(t, sig, sig2)
}
