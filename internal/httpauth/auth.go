package httpauth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/skycoin/dmsg/cipher"
)

// Nonce is used to sign requests in order to avoid replay attack
type Nonce uint64

func (n Nonce) String() string { return fmt.Sprintf("%d", n) }

// Auth holds authentication mandatory values
type Auth struct {
	Key   cipher.PubKey
	Nonce Nonce
	Sig   cipher.Sig
}

// AuthFromHeaders attempts to extract auth from request header
func AuthFromHeaders(hdr http.Header) (*Auth, error) {
	a := &Auth{}
	var v string

	if v = hdr.Get("SW-Public"); v == "" {
		return nil, errors.New("SW-Public missing")
	}
	key := cipher.PubKey{}
	if err := key.UnmarshalText([]byte(v)); err != nil {
		return nil, fmt.Errorf("Error parsing SW-Public: %s", err.Error())
	}
	a.Key = key

	if v = hdr.Get("SW-Sig"); v == "" {
		return nil, errors.New("SW-Sig missing")
	}
	sig := cipher.Sig{}
	if err := sig.UnmarshalText([]byte(v)); err != nil {
		return nil, fmt.Errorf("Error parsing SW-Sig:'%s': %s", v, err.Error())
	}
	a.Sig = sig

	nonceStr := hdr.Get("SW-Nonce")
	if nonceStr == "" {
		return nil, errors.New("SW-Nonce missing")
	}
	nonceUint, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		if numErr, ok := err.(*strconv.NumError); ok {
			return nil, fmt.Errorf("Error parsing SW-Nonce: %s", numErr.Err.Error())
		}

		return nil, fmt.Errorf("Error parsing SW-Nonce: %s", err.Error())
	}
	a.Nonce = Nonce(nonceUint)

	return a, nil
}

// Verify verifies signature of a payload.
func (a *Auth) Verify(in []byte) error {
	return Verify(in, a.Nonce, a.Key, a.Sig)
}

// PayloadWithNonce returns the concatenation of payload and nonce.
func PayloadWithNonce(payload []byte, nonce Nonce) []byte {
	return []byte(fmt.Sprintf("%s%d", string(payload), nonce))
}

// Sign signs the Hash of payload and nonce
func Sign(payload []byte, nonce Nonce, sec cipher.SecKey) (cipher.Sig, error) {
	return cipher.SignPayload(PayloadWithNonce(payload, nonce), sec)
}

// Verify verifies the signature of the hash of payload and nonce
func Verify(payload []byte, nonce Nonce, pub cipher.PubKey, sig cipher.Sig) error {
	return cipher.VerifyPubKeySignedPayload(pub, sig, PayloadWithNonce(payload, nonce))
}
