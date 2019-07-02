package noise

import (
	"io"

	"github.com/flynn/noise"
	"github.com/skycoin/skycoin/src/cipher"
)

// Secp256k1 implements `noise.DHFunc`.
type Secp256k1 struct{}

// GenerateKeypair helps to implement `noise.DHFunc`.
func (Secp256k1) GenerateKeypair(_ io.Reader) (noise.DHKey, error) {
	pk, sk := cipher.GenerateKeyPair()
	return noise.DHKey{
		Private: sk[:],
		Public:  pk[:],
	}, nil
}

// DH helps to implement `noise.DHFunc`.
func (Secp256k1) DH(sk, pk []byte) []byte {
	return append(
		cipher.MustECDH(cipher.MustNewPubKey(pk), cipher.MustNewSecKey(sk)),
		byte(0))
}

// DHLen helps to implement `noise.DHFunc`.
func (Secp256k1) DHLen() int {
	return 33
}

// DHName helps to implement `noise.DHFunc`.
func (Secp256k1) DHName() string {
	return "Secp256k1"
}
