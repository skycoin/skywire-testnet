package messaging

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const handshakeTimeout = time.Second

func TestHandshakeFrame(t *testing.T) {
	initPK, initSK := cipher.GenerateKeyPair()
	respPK, respSK := cipher.GenerateKeyPair()
	frame := newHandshakeFrame(initPK, respPK)

	p, err := frame.toBinary()
	require.NoError(t, err)
	require.Len(t, p, 3+33+33+16)

	s1, err := frame.signature(respSK)
	require.NoError(t, err)
	require.NoError(t, frame.verifySignature(s1, sig1))

	frame.Sig1 = s1
	p, err = frame.toBinary()
	require.NoError(t, err)
	require.Len(t, p, 3+33+33+16+65)

	s2, err := frame.signature(initSK)
	require.NoError(t, err)
	require.NoError(t, frame.verifySignature(s2, sig2))

	frame.Sig2 = s2
	p, err = frame.toBinary()
	require.NoError(t, err)
	require.Len(t, p, 3+33+33+16+65+65)
}

func TestHandshake(t *testing.T) {
	initPK, initSK := cipher.GenerateKeyPair()
	respPK, respSK := cipher.GenerateKeyPair()
	initConf := &LinkConfig{
		Public: initPK,
		Secret: initSK,
		Remote: respPK,
	}
	respConf := &LinkConfig{
		Public: respPK,
		Secret: respSK,
	}

	initHandshake := initiatorHandshake(initConf)
	respHandshake := responderHandshake(respConf)

	initNet, respNet := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- respHandshake.Do(json.NewDecoder(respNet), json.NewEncoder(respNet), handshakeTimeout)
	}()

	err := initHandshake.Do(json.NewDecoder(initNet), json.NewEncoder(initNet), handshakeTimeout)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, initPK, respConf.Remote)
}

func TestHandshakeInvalidResponder(t *testing.T) {
	initPK, initSK := cipher.GenerateKeyPair()
	respPK, respSK := cipher.GenerateKeyPair()
	anotherPK, _ := cipher.GenerateKeyPair()
	initConf := &LinkConfig{
		Public: initPK,
		Secret: initSK,
		Remote: anotherPK,
	}
	respConf := &LinkConfig{
		Public: respPK,
		Secret: respSK,
	}

	initHandshake := initiatorHandshake(initConf)
	respHandshake := responderHandshake(respConf)

	initNet, respNet := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- respHandshake.Do(json.NewDecoder(respNet), json.NewEncoder(respNet), handshakeTimeout)
	}()

	err := initHandshake.Do(json.NewDecoder(initNet), json.NewEncoder(initNet), handshakeTimeout)
	require.Error(t, err)
	assert.Equal(t, "invalid sig1: Recovered pubkey does not match pubkey", err.Error())

	err = <-errCh
	require.Error(t, err)
	assert.Equal(t, "handshake failed", err.Error())
}

func TestHandshakeInvalidInitiator(t *testing.T) {
	initPK, _ := cipher.GenerateKeyPair()
	respPK, respSK := cipher.GenerateKeyPair()
	_, anotherSK := cipher.GenerateKeyPair()
	initConf := &LinkConfig{
		Public: initPK,
		Secret: anotherSK,
		Remote: respPK,
	}
	respConf := &LinkConfig{
		Public: respPK,
		Secret: respSK,
	}

	initHandshake := initiatorHandshake(initConf)
	respHandshake := responderHandshake(respConf)

	initNet, respNet := net.Pipe()
	errCh := make(chan error)
	go func() {
		errCh <- respHandshake.Do(json.NewDecoder(respNet), json.NewEncoder(respNet), handshakeTimeout)
	}()

	err := initHandshake.Do(json.NewDecoder(initNet), json.NewEncoder(initNet), handshakeTimeout)
	require.Error(t, err)
	assert.Equal(t, "handshake failed", err.Error())

	err = <-errCh
	require.Error(t, err)
	assert.Equal(t, "invalid sig2: Recovered pubkey does not match pubkey", err.Error())
}
