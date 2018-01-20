package factory

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"io"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

func init() {
	ops[OP_REG] = &sync.Pool{
		New: func() interface{} {
			return new(reg)
		},
	}
	resps[OP_REG] = &sync.Pool{
		New: func() interface{} {
			return new(regResp)
		},
	}

	ops[OP_REG_KEY] = &sync.Pool{
		New: func() interface{} {
			return new(regWithKey)
		},
	}
	resps[OP_REG_KEY] = &sync.Pool{
		New: func() interface{} {
			return new(regWithKeyResp)
		},
	}

	ops[OP_REG_SIG] = &sync.Pool{
		New: func() interface{} {
			return new(regCheckSig)
		},
	}
	resps[OP_REG_SIG] = &sync.Pool{
		New: func() interface{} {
			return new(regResp)
		},
	}
}

type reg struct {
}

func (reg *reg) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if conn.IsKeySet() {
		conn.GetContextLogger().Infof("reg %s already", conn.key.Hex())
		return
	}
	key, _ := cipher.GenerateKeyPair()
	conn.SetKey(key)
	conn.SetContextLogger(conn.GetContextLogger().WithField("pubkey", key.Hex()))
	f.register(key, conn)
	r = &regResp{PubKey: key}
	return
}

type regResp struct {
	PubKey cipher.PubKey
}

func (resp *regResp) Run(conn *Connection) (err error) {
	conn.SetKey(resp.PubKey)
	conn.SetContextLogger(conn.GetContextLogger().WithField("pubkey", resp.PubKey.Hex()))
	return
}

const (
	publicKey = iota
	randomBytes
)

type RegVersion int

const (
	regWithKeyVersion RegVersion = iota
	RegWithKeyAndEncryptionVersion
)

type regWithKey struct {
	PublicKey cipher.PubKey
	Context   map[string]string
	Version   RegVersion
}

func (reg *regWithKey) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if conn.IsKeySet() {
		conn.GetContextLogger().Infof("reg %s already", conn.key.Hex())
		return
	}
	for k, v := range reg.Context {
		conn.StoreContext(k, v)
	}
	conn.StoreContext(publicKey, reg.PublicKey)
	if reg.Version == RegWithKeyAndEncryptionVersion {
		sc := f.GetDefaultSeedConfig()
		if sc == nil {
			err = errors.New("GetDefaultSeedConfig is nil")
			return
		}
		n := cipher.RandByte(64)
		hash := cipher.SumSHA256(n)
		conn.StoreContext(randomBytes, hash)
		resp := &regWithKeyResp{
			Num:       make([]byte, aes.BlockSize),
			PublicKey: sc.publicKey,
			Version:   reg.Version,
			Hash:      hash,
		}
		if _, err = io.ReadFull(rand.Reader, resp.Num); err != nil {
			return
		}
		err = conn.SetCrypto(sc.publicKey, sc.secKey, reg.PublicKey, resp.Num)
		if err != nil {
			return
		}

		err = conn.writeOPReq(OP_REG_KEY|RESP_PREFIX,
			resp)
		return
	}
	n := cipher.RandByte(64)
	conn.StoreContext(randomBytes, n)
	r = &regWithKeyResp{Num: n}
	return
}

type regWithKeyResp struct {
	Num       []byte
	Hash      cipher.SHA256
	PublicKey cipher.PubKey
	Version   RegVersion
}

func (resp *regWithKeyResp) Run(conn *Connection) (err error) {
	if resp.Version == RegWithKeyAndEncryptionVersion {
		k, ok := conn.context.Load(publicKey)
		if !ok {
			err = errors.New("public key not found")
			return
		}
		pk, ok := k.(cipher.PubKey)
		if !ok {
			err = errors.New("public key invalid")
			return
		}
		tpk := resp.PublicKey
		t := conn.GetTargetKey()
		if t != EMPATY_PUBLIC_KEY && t != tpk {
			tpk = t
		}
		err = conn.SetCrypto(pk, conn.GetSecKey(), tpk, resp.Num)
		if err != nil {
			return
		}
		sig := cipher.SignHash(resp.Hash, conn.GetSecKey())
		err = conn.writeOPResp(OP_REG_SIG, &regCheckSig{
			Sig:     sig,
			Version: resp.Version,
		})
		conn.SetKey(pk)
		return
	}
	sk := conn.GetSecKey()
	hash := cipher.SumSHA256(resp.Num)
	sig := cipher.SignHash(hash, sk)
	err = conn.writeOP(OP_REG_SIG, &regCheckSig{Sig: sig})
	return
}

type regCheckSig struct {
	Sig     cipher.Sig
	Version RegVersion
}

func (reg *regCheckSig) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if conn.IsKeySet() {
		conn.GetContextLogger().Infof("reg %s already", conn.key.Hex())
		return
	}
	k, ok := conn.context.Load(publicKey)
	if !ok {
		err = errors.New("public key not found")
		return
	}
	pk, ok := k.(cipher.PubKey)
	if !ok {
		err = errors.New("public key invalid")
		return
	}
	if reg.Version == RegWithKeyAndEncryptionVersion && conn.GetCrypto() != nil {
		n, ok := conn.context.Load(randomBytes)
		if !ok {
			err = errors.New("hash not found")
			return
		}
		hash, ok := n.(cipher.SHA256)
		if !ok {
			err = errors.New("hash is invalid")
			return
		}
		err = cipher.VerifySignature(pk, reg.Sig, hash)
		if err != nil {
			return
		}
		goto OK
	} else {
		n, ok := conn.context.Load(randomBytes)
		if !ok {
			err = errors.New("randomBytes not found")
			return
		}
		hash := cipher.SumSHA256(n.([]byte))
		err = cipher.VerifySignature(pk, reg.Sig, hash)
		if err != nil {
			return
		}
	}
	r = &regResp{PubKey: pk}
OK:
	conn.SetKey(pk)
	conn.SetContextLogger(conn.GetContextLogger().WithField("pubkey", pk.Hex()))
	if conn.IsTCP() {
		f.register(pk, conn)
	}
	return
}
