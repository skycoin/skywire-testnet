package dmsg

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/skycoin/skywire/internal/ioutil"

	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	// Type returns the transport type string.
	Type = "dmsg"

	hsTimeout     = time.Second * 10
	tpBufCap      = math.MaxUint16
	tpBufFrameCap = math.MaxUint8
	tpAckCap      = math.MaxUint8
	acceptChSize  = 20
	headerLen     = 5 // fType(1 byte), chID(2 byte), payLen(2 byte)
)

func isInitiatorID(tpID uint16) bool { return tpID%2 == 0 }

func randID(initiator bool) uint16 {
	var id uint16
	for {
		id = binary.BigEndian.Uint16(cipher.RandByte(2))
		if initiator && id%2 == 0 || !initiator && id%2 != 0 {
			return id
		}
	}
}

var serveCount int64

func incrementServeCount() int64 { return atomic.AddInt64(&serveCount, 1) }
func decrementServeCount() int64 { return atomic.AddInt64(&serveCount, -1) }

// FrameType represents the frame type.
type FrameType byte

func (ft FrameType) String() string {
	var names = []string{
		RequestType: "REQUEST",
		AcceptType:  "ACCEPT",
		CloseType:   "CLOSE",
		FwdType:     "FWD",
		AckType:     "ACK",
	}
	if int(ft) >= len(names) {
		return fmt.Sprintf("UNKNOWN:%d", ft)
	}
	return names[ft]
}

// Frame types.
const (
	RequestType = FrameType(0x1)
	AcceptType  = FrameType(0x2)
	CloseType   = FrameType(0x3)
	FwdType     = FrameType(0xa)
	AckType     = FrameType(0xb)
)

// Frame is the dmsg data unit.
type Frame []byte

// type RequestFrame Frame
// type AcceptFrame Frame
// type CloseFrame Frame
// type FwdFrame Frame
// type AccFrame Frame

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

// String implements io.Stringer
func (f Frame) String() string {
	var p string
	switch f.Type() {
	case FwdType, AckType:
		p = fmt.Sprintf("<seq:%d>", ioutil.DecodeUint16Seq(f.Pay()))
	}
	return fmt.Sprintf("<type:%s><id:%d><size:%d>%s", f.Type(), f.TpID(), f.PayLen(), p)
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

type writeError struct{ error }

func (e *writeError) Error() string { return "write error: " + e.error.Error() }

func isWriteError(err error) bool {
	_, ok := err.(*writeError)
	return ok
}

func writeFrame(w io.Writer, f Frame) error {
	_, err := w.Write(f)
	if err != nil {
		return &writeError{err}
	}
	return nil
}

func writeFwdFrame(w io.Writer, id uint16, seq ioutil.Uint16Seq, p []byte) error {
	return writeFrame(w, MakeFrame(FwdType, id, append(seq.Encode(), p...)))
}

func writeCloseFrame(w io.Writer, id uint16, reason byte) error {
	return writeFrame(w, MakeFrame(CloseType, id, []byte{reason}))
}

func combinePKs(initPK, respPK cipher.PubKey) []byte {
	return append(initPK[:], respPK[:]...)
}

func splitPKs(b []byte) (initPK, respPK cipher.PubKey, ok bool) {
	const pkLen = 33

	if len(b) != pkLen*2 {
		ok = false
		return
	}
	copy(initPK[:], b[:pkLen])
	copy(respPK[:], b[pkLen:])
	return initPK, respPK, true
}
