// Package cipher implements common golang encoding interfaces for
// github.com/skycoin/skycoin/src/cipher
package cipher

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

func init() {
	cipher.DebugLevel2 = false // DebugLevel2 causes ECDH to be really slow
}

// GenerateKeyPair creates key pair
func GenerateKeyPair() (PubKey, SecKey) {
	pk, sk := cipher.GenerateKeyPair()
	return PubKey(pk), SecKey(sk)
}

// GenerateDeterministicKeyPair generates deterministic key pair
func GenerateDeterministicKeyPair(seed []byte) (PubKey, SecKey, error) {
	pk, sk, err := cipher.GenerateDeterministicKeyPair(seed)
	return PubKey(pk), SecKey(sk), err
}

// NewPubKey converts []byte to a PubKey
func NewPubKey(b []byte) (PubKey, error) {
	pk, err := cipher.NewPubKey(b)
	return PubKey(pk), err
}

// PubKey is a wrapper type for cipher.PubKey that implements common
// golang interfaces.
type PubKey cipher.PubKey

// Hex returns a hex encoded PubKey string
func (pk PubKey) Hex() string {
	return cipher.PubKey(pk).Hex()
}

// Null returns true if PubKey is the null PubKey
func (pk PubKey) Null() bool {
	return cipher.PubKey(pk).Null()
}

// String implements fmt.Stringer for PubKey. Returns Hex representation.
func (pk PubKey) String() string {
	return pk.Hex()
}

// Set implements pflag.Value for PubKey.
func (pk *PubKey) Set(s string) error {
	cPK, err := cipher.PubKeyFromHex(s)
	if err != nil {
		return err
	}
	*pk = PubKey(cPK)
	return nil
}

// Type implements pflag.Value for PubKey.
func (pk PubKey) Type() string {
	return "cipher.PubKey"
}

// MarshalText implements encoding.TextMarshaler.
func (pk PubKey) MarshalText() ([]byte, error) {
	return []byte(pk.Hex()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (pk *PubKey) UnmarshalText(data []byte) error {
	if bytes.Count(data, []byte{48}) == len(data) {
		return nil
	}

	dPK, err := cipher.PubKeyFromHex(string(data))
	if err == nil {
		*pk = PubKey(dPK)
	}
	return err
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (pk PubKey) MarshalBinary() ([]byte, error) {
	return pk[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (pk *PubKey) UnmarshalBinary(data []byte) error {
	dPK, err := cipher.NewPubKey(data)
	if err == nil {
		*pk = PubKey(dPK)
	}
	return err
}

// PubKeys represents a slice of PubKeys.
type PubKeys []PubKey

// String implements stringer for PubKeys.
func (p PubKeys) String() string {
	res := "public keys:\n"
	for _, pk := range p {
		res += fmt.Sprintf("\t%s\n", pk)
	}
	return res
}

// Set implements pflag.Value for PubKeys.
func (p *PubKeys) Set(list string) error {
	*p = PubKeys{}
	for _, s := range strings.Split(list, ",") {
		var pk PubKey
		if err := pk.Set(strings.TrimSpace(s)); err != nil {
			return err
		}
		*p = append(*p, pk)
	}
	return nil
}

// Type implements pflag.Value for PubKeys.
func (p PubKeys) Type() string {
	return "cipher.PubKeys"
}

// SecKey is a wrapper type for cipher.SecKey that implements common
// golang interfaces.
type SecKey cipher.SecKey

// Hex returns a hex encoded SecKey string
func (sk SecKey) Hex() string {
	return cipher.SecKey(sk).Hex()
}

// Null returns true if SecKey is the null SecKey.
func (sk SecKey) Null() bool {
	return cipher.SecKey(sk).Null()
}

// String implements fmt.Stringer for SecKey. Returns Hex representation.
func (sk SecKey) String() string {
	return sk.Hex()
}

// Set implements pflag.Value for SecKey.
func (sk *SecKey) Set(s string) error {
	cSK, err := cipher.SecKeyFromHex(s)
	if err != nil {
		return err
	}
	*sk = SecKey(cSK)
	return nil
}

// Type implements pflag.Value for SecKey.
func (sk *SecKey) Type() string {
	return "cipher.SecKey"
}

// MarshalText implements encoding.TextMarshaler.
func (sk SecKey) MarshalText() ([]byte, error) {
	return []byte(sk.Hex()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (sk *SecKey) UnmarshalText(data []byte) error {
	dSK, err := cipher.SecKeyFromHex(string(data))
	if err == nil {
		*sk = SecKey(dSK)
	}
	return err
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (sk SecKey) MarshalBinary() ([]byte, error) {
	return sk[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (sk *SecKey) UnmarshalBinary(data []byte) error {
	dSK, err := cipher.NewSecKey(data)
	if err == nil {
		*sk = SecKey(dSK)
	}
	return err
}

// PubKey recovers the public key for a secret key
func (sk SecKey) PubKey() (PubKey, error) {
	pk, err := cipher.PubKeyFromSecKey(cipher.SecKey(sk))
	return PubKey(pk), err
}

// Sig is a wrapper type for cipher.Sig that implements common golang interfaces.
type Sig cipher.Sig

// Hex returns a hex encoded Sig string
func (sig Sig) Hex() string {
	return cipher.Sig(sig).Hex()
}

// String implements fmt.Stringer for Sig. Returns Hex representation.
func (sig Sig) String() string {
	return sig.Hex()
}

// Null returns true if Sig is a null Sig
func (sig Sig) Null() bool {
	return sig == Sig{}
}

// MarshalText implements encoding.TextMarshaler.
func (sig Sig) MarshalText() ([]byte, error) {
	return []byte(sig.Hex()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (sig *Sig) UnmarshalText(data []byte) error {
	dSig, err := cipher.SigFromHex(string(data))
	if err == nil {
		*sig = Sig(dSig)
	}
	return err
}

// SignPayload creates Sig for payload using SHA256
func SignPayload(payload []byte, sec SecKey) (Sig, error) {
	sig, err := cipher.SignHash(cipher.SumSHA256(payload), cipher.SecKey(sec))
	return Sig(sig), err
}

// VerifyPubKeySignedPayload verifies that SHA256 hash of the payload was signed by PubKey
func VerifyPubKeySignedPayload(pubkey PubKey, sig Sig, payload []byte) error {
	return cipher.VerifyPubKeySignedHash(cipher.PubKey(pubkey), cipher.Sig(sig), cipher.SumSHA256(payload))
}

// RandByte returns rand N bytes
func RandByte(n int) []byte {
	return cipher.RandByte(n)
}

// SHA256 is a wrapper type for cipher.SHA256 that implements common
// golang interfaces.
type SHA256 cipher.SHA256

// SHA256FromBytes converts []byte to SHA256
func SHA256FromBytes(b []byte) (SHA256, error) {
	h, err := cipher.SHA256FromBytes(b)
	return SHA256(h), err
}

// SumSHA256 sum sha256
func SumSHA256(b []byte) SHA256 {
	return SHA256(cipher.SumSHA256(b))
}
