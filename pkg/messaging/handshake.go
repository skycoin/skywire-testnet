package messaging

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

// HandshakeNonceSize defines size of the nonce used in handshake.
const HandshakeNonceSize = 16

type sigType = byte

const (
	sig1 sigType = iota
	sig2
)

type handshakeFrame struct {
	Version   string        `json:"version"`
	Initiator cipher.PubKey `json:"initiator"`
	Responder cipher.PubKey `json:"responder"`
	Nonce     string        `json:"nonce"`
	Sig1      cipher.Sig    `json:"sig1,omitempty"`
	Sig2      cipher.Sig    `json:"sig2,omitempty"`
	Accepted  bool          `json:"accepted,omitempty"`
}

func newHandshakeFrame(initiator, responder cipher.PubKey) *handshakeFrame {
	nonce := cipher.RandByte(HandshakeNonceSize)
	return &handshakeFrame{
		Version:   "0.1",
		Initiator: initiator,
		Responder: responder,
		Nonce:     hex.EncodeToString(nonce),
	}
}

func (p *handshakeFrame) toBinary() ([]byte, error) {
	nonce, err := hex.DecodeString(p.Nonce)
	if err != nil {
		return nil, err
	}

	buf := append([]byte(p.Version), p.Initiator[:]...)
	buf = append(buf, p.Responder[:]...)
	buf = append(buf, nonce...)

	if p.Sig1.Null() {
		return buf, nil
	}

	buf = append(buf, p.Sig1[:]...)

	if p.Sig2.Null() {
		return buf, nil
	}

	buf = append(buf, p.Sig2[:]...)

	return buf, nil
}

func (p *handshakeFrame) signature(secKey cipher.SecKey) (sig cipher.Sig, err error) {
	var bPayload []byte
	bPayload, err = p.toBinary()
	if err != nil {
		return
	}

	sig, err = cipher.SignPayload(bPayload, secKey)
	if err != nil {
		return
	}

	return sig, err
}

func (p *handshakeFrame) verifySignature(sig cipher.Sig, sigType sigType) error {
	var pk cipher.PubKey
	if sigType == sig1 {
		pk = p.Responder
	} else {
		pk = p.Initiator
	}

	bPayload, err := p.toBinary()
	if err != nil {
		return err
	}

	return cipher.VerifyPubKeySignedPayload(pk, sig, bPayload)
}

// Handshake represents a set of actions that an instance performs to complete a handshake.
type Handshake func(dec *json.Decoder, enc *json.Encoder) error

// Do performs a handshake with a given timeout.
// Non-nil error is returned on failure.
func (handshake Handshake) Do(dec *json.Decoder, enc *json.Encoder, timeout time.Duration) (err error) {
	done := make(chan struct{})
	go func() {
		err = handshake(dec, enc)
		close(done)
	}()
	select {
	case <-done:
		return err
	case <-time.After(timeout):
		return ErrHandshakeFailed
	}
}

func initiatorHandshake(c *LinkConfig) Handshake {
	return func(dec *json.Decoder, enc *json.Encoder) error {
		frame := newHandshakeFrame(c.Public, c.Remote)
		if err := enc.Encode(frame); err != nil {
			return err
		}

		var resFrame *handshakeFrame
		var err error
		if err := dec.Decode(&resFrame); err != nil {
			return err
		}

		if err := frame.verifySignature(resFrame.Sig1, sig1); err != nil {
			return fmt.Errorf("invalid sig1: %s", err)
		}

		if resFrame.Sig2, err = resFrame.signature(c.Secret); err != nil {
			return fmt.Errorf("failed to make sig2: %s", err)
		}

		if err := enc.Encode(resFrame); err != nil {
			return err
		}

		if err := dec.Decode(&resFrame); err != nil {
			return err
		}

		if !resFrame.Accepted {
			return errors.New("rejected")
		}

		return nil
	}
}

func responderHandshake(c *LinkConfig) Handshake {
	return func(dec *json.Decoder, enc *json.Encoder) error {
		var frame *handshakeFrame
		var err error

		if err := dec.Decode(&frame); err != nil {
			return err
		}

		if frame.Sig1, err = frame.signature(c.Secret); err != nil {
			return fmt.Errorf("failed to make sig1: %s", err)
		}

		if err := enc.Encode(frame); err != nil {
			return err
		}

		var resFrame *handshakeFrame
		if err := dec.Decode(&resFrame); err != nil {
			return err
		}

		if err := frame.verifySignature(resFrame.Sig2, sig2); err != nil {
			return fmt.Errorf("invalid sig2: %s", err)
		}

		resFrame.Accepted = true
		if err := enc.Encode(resFrame); err != nil {
			return err
		}

		c.Remote = frame.Initiator
		return nil
	}
}
