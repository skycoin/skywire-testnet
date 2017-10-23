package factory

import (
	"errors"
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

type regWithKey struct {
	PublicKey cipher.PubKey
	Context   map[string]string
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
	n := cipher.RandByte(64)
	conn.StoreContext(randomBytes, n)
	r = &regWithKeyResp{Num: n}
	return
}

type regWithKeyResp struct {
	Num []byte
}

func (resp *regWithKeyResp) Run(conn *Connection) (err error) {
	sk := conn.GetSecKey()
	hash := cipher.SumSHA256(resp.Num)
	sig := cipher.SignHash(hash, sk)
	err = conn.writeOP(OP_REG_SIG, &regCheckSig{Sig: sig})
	return
}

type regCheckSig struct {
	Sig cipher.Sig
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
	n, ok := conn.context.Load(randomBytes)
	if !ok {
		err = errors.New("randomBytes not found")
		return
	}
	hash := cipher.SumSHA256(n.([]byte))
	err = cipher.VerifySignature(pk, reg.Sig, hash)
	if err == nil {
		conn.SetKey(pk)
		conn.SetContextLogger(conn.GetContextLogger().WithField("pubkey", pk.Hex()))
		f.register(pk, conn)
		r = &regResp{PubKey: pk}
	}
	return
}
