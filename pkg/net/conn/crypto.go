package conn

import (
	"crypto/aes"
	cipher2 "crypto/cipher"
	"errors"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"io"
	"sync"
	"sync/atomic"
)

type Crypto struct {
	key     cipher.PubKey
	secKey  cipher.SecKey
	target  cipher.PubKey
	block   atomic.Value
	es      cipher2.Stream
	esMutex sync.Mutex
	ds      cipher2.Stream
	dsMutex sync.Mutex
}

func NewCrypto(key cipher.PubKey, secKey cipher.SecKey) *Crypto {
	return &Crypto{
		key:    key,
		secKey: secKey,
	}
}

func (c *Crypto) SetTargetKey(target cipher.PubKey) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("SetTargetKey recovered err %v", e)
		}
	}()
	c.target = target
	ecdh := cipher.ECDH(target, c.secKey)
	b, err := aes.NewCipher(ecdh)
	c.block.Store(b)
	return
}

func (c *Crypto) Init(iv []byte) (err error) {
	block := c.block.Load()
	if block == nil {
		err = errors.New("call SetTargetKey first")
		return
	}

	c.esMutex.Lock()
	c.es = cipher2.NewCFBEncrypter(block.(cipher2.Block), iv)
	c.esMutex.Unlock()
	c.dsMutex.Lock()
	c.ds = cipher2.NewCFBDecrypter(block.(cipher2.Block), iv)
	c.dsMutex.Unlock()
	return
}

func (c *Crypto) Encrypt(data []byte) (err error) {
	block := c.block.Load()
	if block == nil {
		err = errors.New("call SetTargetKey first")
		return
	}

	c.esMutex.Lock()
	c.es.XORKeyStream(data, data)
	c.esMutex.Unlock()
	return
}

func (c *Crypto) Decrypt(data []byte) (err error) {
	block := c.block.Load()
	if block == nil {
		err = errors.New("call SetTargetKey first")
		return
	}

	c.dsMutex.Lock()
	c.ds.XORKeyStream(data, data)
	c.dsMutex.Unlock()
	return
}

type CryptoGetter interface {
	GetCrypto() *Crypto
}

type CryptoReader struct {
	rd io.Reader
	cg CryptoGetter
}

func NewCryptoReader(rd io.Reader, getter CryptoGetter) *CryptoReader {
	return &CryptoReader{
		rd: rd,
		cg: getter,
	}
}

func (cr *CryptoReader) Read(p []byte) (n int, err error) {
	n, err = cr.rd.Read(p)
	if err != nil || n == 0 {
		return
	}
	crypto := cr.cg.GetCrypto()
	if crypto == nil {
		return
	}
	err = crypto.Decrypt(p[:n])
	return
}
