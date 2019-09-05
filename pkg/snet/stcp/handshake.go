package stcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	cipher2 "github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
)

const (
	HandshakeTimeout   = time.Second * 10
	HandshakeNonceSize = 16
)

type HandshakeError string

func (err HandshakeError) Error() string {
	return fmt.Sprintln("stcp handshake failed:", string(err))
}

func IsHandshakeError(err error) bool {
	_, ok := err.(HandshakeError)
	return ok
}

func handshakeErrorMiddleware(origin Handshake) Handshake {
	return func(conn net.Conn, deadline time.Time) (lAddr, rAddr dmsg.Addr, err error) {
		if err = conn.SetDeadline(deadline); err != nil {
			return
		}
		if lAddr, rAddr, err = origin(conn, deadline); err != nil {
			err = HandshakeError(err.Error())
		}

		// reset deadline
		_ = conn.SetDeadline(time.Time{}) //nolint:errcheck
		return
	}
}

type Handshake func(conn net.Conn, deadline time.Time) (lAddr, rAddr dmsg.Addr, err error)

func InitiatorHandshake(lSK cipher.SecKey, localAddr, remoteAddr dmsg.Addr) Handshake {
	return handshakeErrorMiddleware(func(conn net.Conn, deadline time.Time) (lAddr, rAddr dmsg.Addr, err error) {
		var f1 Frame1
		if f1, err = readFrame1(conn); err != nil {
			return
		}
		f2 := Frame2{SrcAddr: localAddr, DstAddr: remoteAddr, Nonce: f1.Nonce}
		if err = f2.Sign(lSK); err != nil {
			return
		}
		if err = writeFrame2(conn, f2); err != nil {
			return
		}
		var f3 Frame3
		if f3, err = readFrame3(conn); err != nil {
			return
		}
		if !f3.OK {
			err = fmt.Errorf("handshake rejected: %s", f3.ErrMsg)
			return
		}
		lAddr = localAddr
		rAddr = remoteAddr
		return
	})
}

func ResponderHandshake(checkF2 func(f2 Frame2) error) Handshake {
	return handshakeErrorMiddleware(func(conn net.Conn, deadline time.Time) (lAddr, rAddr dmsg.Addr, err error) {
		var nonce [HandshakeNonceSize]byte
		copy(nonce[:], cipher.RandByte(HandshakeNonceSize))
		if err = writeFrame1(conn, nonce); err != nil {
			return
		}
		var f2 Frame2
		if f2, err = readFrame2(conn); err != nil {
			return
		}
		if err = f2.Verify(nonce); err != nil {
			return
		}
		if err = checkF2(f2); err != nil {
			_ = writeFrame3(conn, err) // nolint:errcheck
			return
		}
		lAddr = f2.DstAddr
		rAddr = f2.SrcAddr
		err = writeFrame3(conn, nil)
		return
	})
}

// Dst -> Src
type Frame1 struct {
	Nonce [HandshakeNonceSize]byte
}

// Src -> Dst
type Frame2 struct {
	SrcAddr dmsg.Addr
	DstAddr dmsg.Addr
	Nonce   [HandshakeNonceSize]byte
	Sig     cipher.Sig
}

func (f2 *Frame2) Sign(srcSK cipher.SecKey) error {
	pk, err := srcSK.PubKey()
	if err != nil {
		return err
	}
	f2.SrcAddr.PK = pk
	f2.Sig = cipher.Sig{}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(f2); err != nil {
		return err
	}
	sig, err := cipher.SignPayload(b.Bytes(), srcSK)
	if err != nil {
		return err
	}
	f2.Sig = sig
	fmt.Println("SIGN! len(b.Bytes)", len(b.Bytes()), cipher2.SumSHA256(b.Bytes()).Hex())
	return nil
}

func (f2 Frame2) Verify(nonce [HandshakeNonceSize]byte) error {
	if f2.Nonce != nonce {
		return errors.New("unexpected nonce")
	}

	sig := f2.Sig
	f2.Sig = cipher.Sig{}

	//cipher2.PubKeyFromSig(cipher2.Sig(sig))

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(f2); err != nil {
		return err
	}
	hash := cipher.SumSHA256(b.Bytes())
	rPK, err := cipher2.PubKeyFromSig(cipher2.Sig(sig), cipher2.SHA256(hash))
	fmt.Println("VERIFY! len(b.Bytes)", len(b.Bytes()), cipher2.SHA256(hash).Hex(), "recovered:", rPK, err, "expected:", f2.SrcAddr.PK)

	return cipher.VerifyPubKeySignedPayload(f2.SrcAddr.PK, sig, b.Bytes())
}

type Frame3 struct {
	OK     bool
	ErrMsg string
}

func writeFrame1(w io.Writer, nonce [HandshakeNonceSize]byte) error {
	print("writeF1", Frame1{Nonce: nonce})
	return json.NewEncoder(w).Encode(Frame1{Nonce: nonce})
}

func readFrame1(r io.Reader) (Frame1, error) {
	var f1 Frame1
	err := json.NewDecoder(r).Decode(&f1)
	print("readF1", f1)
	return f1, err
}

func writeFrame2(w io.Writer, f2 Frame2) error {
	print("writeF2", f2)
	return json.NewEncoder(w).Encode(f2)
}

func readFrame2(r io.Reader) (Frame2, error) {
	var f2 Frame2
	err := json.NewDecoder(r).Decode(&f2)
	print("readF2", f2)
	return f2, err
}

func writeFrame3(w io.Writer, err error) error {
	f3 := Frame3{OK: true}
	if err != nil {
		f3.OK = false
		f3.ErrMsg = err.Error()
	}
	print("writeF3", f3)
	return json.NewEncoder(w).Encode(f3)
}

func readFrame3(r io.Reader) (Frame3, error) {
	var f3 Frame3
	err := json.NewDecoder(r).Decode(&f3)
	print("readF3", f3)
	return f3, err
}

func print(prefix string, v interface{}) {
	b, _ := json.MarshalIndent(v, "", "\t") //nolint:errcheck
	fmt.Println(prefix+":", string(b))
}
