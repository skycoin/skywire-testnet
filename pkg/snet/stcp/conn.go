package stcp

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"io"
	"net"
	"time"
)

const (
	nonceSize = 16
)

type F1 struct {
	Nonce [nonceSize]byte
}

type F2 struct {
	SrcPK   cipher.PubKey
	SrcPort uint16
	DstPK   cipher.PubKey
	DstPort uint16
	Nonce   [nonceSize]byte
	Sig     cipher.Sig
}

func (f2 *F2) Sign(srcPK cipher.PubKey, srcSK cipher.SecKey) error {
	f2.SrcPK = srcPK
	f2.Sig = cipher.Sig{}

	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(f2); err != nil {
		return err
	}
	sig, err := cipher.SignPayload(b.Bytes(), srcSK)
	if err != nil {
		return err
	}
	f2.Sig = sig
	return nil
}

func (f2 F2) Verify(dstPK cipher.PubKey, nonce [nonceSize]byte) error {
	if f2.DstPK != dstPK {
		return errors.New("unexpected destination public key")
	}
	if f2.Nonce != nonce {
		return errors.New("unexpected nonce")
	}

	sig := f2.Sig
	f2.Sig = cipher.Sig{}

	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(f2); err != nil {
		return err
	}
	return cipher.VerifyPubKeySignedPayload(f2.SrcPK, sig, b.Bytes())
}

func writeFrame1(w io.Writer, nonce [nonceSize]byte) error {
	return gob.NewEncoder(w).Encode(F1{Nonce: nonce})
}

func readFrame1(r io.Reader) (F1, error) {
	var f1 F1
	err := gob.NewDecoder(r).Decode(&f1)
	return f1, err
}

// lPK(33):lPort(2):rPK(33):rPort(2):sig(65)
func writeFrame2(w io.Writer, f2 F2) error {
	return gob.NewEncoder(w).Encode(f2)
}

func readFrame2(r io.Reader, dstPK cipher.PubKey, nonce [nonceSize]byte) (F2, error) {
	var f2 F2
	if err := gob.NewDecoder(r).Decode(&f2); err != nil {
		return F2{}, err
	}
	return f2, f2.Verify(dstPK, nonce)
}

type InitHS func(conn net.Conn, deadline time.Time) error

func NewInitHS(lSK cipher.SecKey, lPK, rPK cipher.PubKey, lPort, rPort uint16) InitHS {
	return func(conn net.Conn, deadline time.Time) error {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
		f1, err := readFrame1(conn)
		if err != nil {
			return err
		}
		f2 := F2{
			SrcPK: lPK,
			SrcPort: lPort,
			DstPK: rPK,
			DstPort: rPort,
			Nonce: f1.Nonce,
		}
		if err := f2.Sign(lPK, lSK); err != nil {
			return err
		}
		return writeFrame2(conn, f2)
	}
}

type RespHS func(conn net.Conn, deadline time.Duration) (rPK cipher.PubKey, rPort uint16, err error)

func NewRespHS() RespHS {
	return func(conn net.Conn, deadline time.Duration) (rPK cipher.PubKey, rPort uint16, err error) {
		return
	}
}

type Conn struct {
	net.Conn
	lAddr dmsg.Addr
	rAddr dmsg.Addr
}

func (c *Conn) LocalAddr() net.Addr {return c.lAddr}
func (c *Conn) RemoteAddr() net.Addr {return c.rAddr}


type Client struct {
	pk   cipher.PubKey
	sk   cipher.SecKey
	addr string
	t    PKTable
}

func (c *Client) Dial(pk cipher.PubKey, port uint16) (*Conn, error) {

}