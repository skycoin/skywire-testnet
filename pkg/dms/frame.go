package dms

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	TpType = "dms"

	hsTimeout  = time.Second * 10
	readBufLen = 10
	headerLen  = 5 // fType(1 byte), chID(2 byte), payLen(2 byte)
)

func isEven(chID uint16) bool { return chID%2 == 0 }

type FrameType byte

const (
	RequestType = FrameType(1)
	AcceptType  = FrameType(2)
	CloseType   = FrameType(3)
	FwdType     = FrameType(10)
)

type Frame []byte

func MakeFrame(ft FrameType, chID uint16, pay []byte) Frame {
	f := make(Frame, headerLen+len(pay))
	f[0] = byte(ft)
	binary.BigEndian.PutUint16(f[1:3], chID)
	binary.BigEndian.PutUint16(f[3:5], uint16(len(pay)))
	copy(f[5:], pay)
	return f
}

func (f Frame) Type() FrameType { return FrameType(f[0]) }
func (f Frame) ChID() uint16    { return binary.BigEndian.Uint16(f[1:3]) }
func (f Frame) PayLen() int     { return int(binary.BigEndian.Uint16(f[3:5])) }
func (f Frame) Pay() []byte     { return f[headerLen:] }

func (f Frame) Disassemble() (ft FrameType, id uint16, p []byte) {
	return f.Type(), f.ChID(), f.Pay()
}

func readFrame(r io.Reader) (Frame, error) {
	fmt.Println("READING FRAME...")
	f := make(Frame, headerLen)
	if _, err := io.ReadFull(r, f); err != nil {
		fmt.Println("READ HEADER:", err)
		return nil, err
	}
	fmt.Println("READ HEADER: OK")
	f = append(f, make([]byte, f.PayLen())...)
	_, err := io.ReadFull(r, f[headerLen:])
	fmt.Println("READ PAYLOAD:", err)
	return f, err
}

func writeFrame(w io.Writer, f Frame) error {
	if _, err := w.Write(f[:headerLen]); err != nil {
		return err
	}
	_, err := w.Write(f[headerLen:])
	//_, err := w.Write(f)
	return err
}

func combinePKs(initPK, respPK cipher.PubKey) []byte {
	return append(initPK[:], respPK[:]...)
}

func splitPKs(b []byte) (initPK, respPK cipher.PubKey, ok bool) {
	pkLen := 33
	if len(b) != pkLen*2 {
		ok = false
		return
	}
	copy(initPK[:], b[:pkLen])
	copy(respPK[:], b[pkLen:])
	return initPK, respPK, true
}
