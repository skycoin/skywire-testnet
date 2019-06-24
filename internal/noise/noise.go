package noise

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/flynn/noise"

	"github.com/skycoin/skywire/pkg/cipher"
)

var log = logging.MustGetLogger("noise")

// Config hold noise parameters.
type Config struct {
	LocalPK   cipher.PubKey // Local instance static public key.
	LocalSK   cipher.SecKey // Local instance static secret key.
	RemotePK  cipher.PubKey // Remote instance static public key.
	Initiator bool          // Whether the local instance initiates the connection.
}

// Noise handles the handshake and the frame's cryptography.
// All operations on Noise are not guaranteed to be thread-safe.
type Noise struct {
	pk   cipher.PubKey
	sk   cipher.SecKey
	init bool

	pattern noise.HandshakePattern
	hs      *noise.HandshakeState
	enc     *noise.CipherState
	dec     *noise.CipherState

	seq             uint32 // sequence number, used as nonce for both encrypting and decrypting
	previousSeq     uint32 // sequence number last decrypted, check in order to avoid reply attacks
	highestPrevious uint32 // highest sequence number received from the other end
	//encN uint32 // counter to inform encrypting CipherState to re-key
	//decN uint32 // counter to inform decrypting CipherState to re-key
}

// New creates a new Noise with:
//	- provided pattern for handshake.
//	- Secp256k1 for the curve.
func New(pattern noise.HandshakePattern, config Config) (*Noise, error) {
	nc := noise.Config{
		CipherSuite: noise.NewCipherSuite(Secp256k1{}, noise.CipherChaChaPoly, noise.HashSHA256),
		Random:      rand.Reader,
		Pattern:     pattern,
		Initiator:   config.Initiator,
		StaticKeypair: noise.DHKey{
			Public:  config.LocalPK[:],
			Private: config.LocalSK[:],
		},
	}
	if !config.RemotePK.Null() {
		nc.PeerStatic = config.RemotePK[:]
	}

	hs, err := noise.NewHandshakeState(nc)
	if err != nil {
		return nil, err
	}
	return &Noise{
		pk:      config.LocalPK,
		sk:      config.LocalSK,
		init:    config.Initiator,
		pattern: pattern,
		hs:      hs,
	}, nil
}

// KKAndSecp256k1 creates a new Noise with:
//	- KK pattern for handshake.
//	- Secp256k1 for the curve.
func KKAndSecp256k1(config Config) (*Noise, error) {
	return New(noise.HandshakeKK, config)
}

// XKAndSecp256k1 creates a new Noise with:
//  - XK pattern for handshake.
//	- Secp256 for the curve.
func XKAndSecp256k1(config Config) (*Noise, error) {
	return New(noise.HandshakeXK, config)
}

// HandshakeMessage generates handshake message for a current handshake state.
func (ns *Noise) HandshakeMessage() (res []byte, err error) {
	if ns.hs.MessageIndex() < len(ns.pattern.Messages)-1 {
		res, _, _, err = ns.hs.WriteMessage(nil, nil)
		return
	}

	res, ns.dec, ns.enc, err = ns.hs.WriteMessage(nil, nil)
	return res, err
}

// ProcessMessage processes a received handshake message and appends the payload.
func (ns *Noise) ProcessMessage(msg []byte) (err error) {
	if ns.hs.MessageIndex() < len(ns.pattern.Messages)-1 {
		_, _, _, err = ns.hs.ReadMessage(nil, msg)
		return
	}

	_, ns.enc, ns.dec, err = ns.hs.ReadMessage(nil, msg)
	return err
}

// LocalStatic returns the local static public key.
func (ns *Noise) LocalStatic() cipher.PubKey {
	return ns.pk
}

// RemoteStatic returns the remote static public key.
func (ns *Noise) RemoteStatic() cipher.PubKey {
	pk, err := cipher.NewPubKey(ns.hs.PeerStatic())
	if err != nil {
		panic(err)
	}
	return cipher.PubKey(pk)
}

// EncryptUnsafe encrypts plaintext without interlocking, should only
// be used with external lock.
func (ns *Noise) EncryptUnsafe(plaintext []byte) []byte {
	ns.seq++
	seq := make([]byte, 4)
	binary.BigEndian.PutUint32(seq, ns.seq)

	return append(seq, ns.enc.Cipher().Encrypt(nil, uint64(ns.seq), nil, plaintext)...)
}

// DecryptUnsafe decrypts ciphertext without interlocking, should only
// be used with external lock.
func (ns *Noise) DecryptUnsafe(ciphertext []byte) ([]byte, error) {
	seq := binary.BigEndian.Uint32(ciphertext[:4])
	if seq <= ns.previousSeq {
		log.Warnf("current seq: %s is not higher than previous one: %s. "+
			"Highest sequence number received so far is: %s", ns.seq, ns.previousSeq, ns.highestPrevious)
	} else {
		if ns.previousSeq > ns.highestPrevious {
			ns.highestPrevious = seq
		}
		ns.previousSeq = seq
	}

	return ns.dec.Cipher().Decrypt(nil, uint64(seq), nil, ciphertext[4:])
}

// HandshakeFinished indicate whether handshake was completed.
func (ns *Noise) HandshakeFinished() bool {
	return ns.hs.MessageIndex() == len(ns.pattern.Messages)
}
