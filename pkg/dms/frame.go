package dms

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	// Type returns the transport type string.
	Type = "dms"

	hsTimeout  = time.Second * 10
	readBufLen = 10
	headerLen  = 5 // fType(1 byte), chID(2 byte), payLen(2 byte)
)

func isEven(chID uint16) bool { return chID%2 == 0 }

// FrameType represents the frame type.
type FrameType byte

func (ft FrameType) String() string {
	var names = []string{
		RequestType: "REQUEST",
		AcceptType:  "ACCEPT",
		CloseType:   "CLOSE",
		SendType:    "SEND",
	}
	if int(ft) >= len(names) {
		return fmt.Sprintf("UNKNOWN:%d", ft)
	}
	return names[ft]
}

// Frame types.
const (
	RequestType = FrameType(1)
	AcceptType  = FrameType(2)
	CloseType   = FrameType(3)
	SendType    = FrameType(10)
)

// Frame is the dms data unit.
type Frame []byte

// MakeFrame creates a new Frame.
func MakeFrame(ft FrameType, chID uint16, pay []byte) Frame {
	f := make(Frame, headerLen+len(pay))
	f[0] = byte(ft)
	binary.BigEndian.PutUint16(f[1:3], chID)
	binary.BigEndian.PutUint16(f[3:5], uint16(len(pay)))
	copy(f[5:], pay)
	return f
}

// Type returns the frame's type.
func (f Frame) Type() FrameType { return FrameType(f[0]) }

// TpID returns the frame's tp_id.
func (f Frame) TpID() uint16 { return binary.BigEndian.Uint16(f[1:3]) }

// PayLen returns the expected payload len.
func (f Frame) PayLen() int { return int(binary.BigEndian.Uint16(f[3:5])) }

// Pay returns the payload.
func (f Frame) Pay() []byte { return f[headerLen:] }

// Disassemble splits the frame into fields.
func (f Frame) Disassemble() (ft FrameType, id uint16, p []byte) {
	return f.Type(), f.TpID(), f.Pay()
}

func readFrame(r io.Reader) (Frame, error) {
	f := make(Frame, headerLen)
	if _, err := io.ReadFull(r, f); err != nil {
		return nil, err
	}
	f = append(f, make([]byte, f.PayLen())...)
	_, err := io.ReadFull(r, f[headerLen:])
	return f, err
}

func writeFrame(w io.Writer, f Frame) error {
	_, err := w.Write(f)
	return err
}

func writeCloseFrame(w io.Writer, id uint16, reason byte) error {
	return writeFrame(w, MakeFrame(CloseType, id, []byte{reason}))
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
