package app2

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/skycoin/skywire/pkg/routing"
)

const (
	HSFrameHeaderLen = 3
	HSFrameProcIDLen = 2
	HSFrameTypeLen   = 1
	HSFramePKLen     = 33
	HSFramePortLen   = 2
)

// HSFrameType identifies the type of a handshake frame.
type HSFrameType byte

const (
	HSFrameTypeDMSGListen HSFrameType = 10 + iota
	HSFrameTypeDMSGListening
	HSFrameTypeDMSGDial
	HSFrameTypeDMSGAccept
	HSFrameTypeStopListening
)

// HSFrame is the data unit for socket connection handshakes between Server and Client.
// It consists of header and body.
//
// Header is a big-endian encoded 3 bytes and is constructed as follows:
// | ProcID (2 bytes) | HSFrameType (1 byte) |
type HSFrame []byte

func newHSFrame(procID ProcID, frameType HSFrameType, bodyLen int) HSFrame {
	hsFrame := make(HSFrame, HSFrameHeaderLen+bodyLen)

	hsFrame.SetProcID(procID)
	hsFrame.SetFrameType(frameType)

	return hsFrame
}

func NewHSFrameDMSGListen(procID ProcID, local routing.Addr) HSFrame {
	hsFrame := newHSFrame(procID, HSFrameTypeDMSGListen, HSFramePKLen+HSFramePortLen)

	copy(hsFrame[HSFrameHeaderLen:], local.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:], uint16(local.Port))

	return hsFrame
}

func NewHSFrameDMSGListening(procID ProcID, local routing.Addr) HSFrame {
	hsFrame := newHSFrame(procID, HSFrameTypeDMSGListening, HSFramePKLen+HSFramePortLen)

	copy(hsFrame[HSFrameHeaderLen:], local.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:], uint16(local.Port))

	return hsFrame
}

func NewHSFrameDSMGDial(procID ProcID, loop routing.Loop) HSFrame {
	hsFrame := newHSFrame(procID, HSFrameTypeDMSGDial, 2*HSFramePKLen+2*HSFramePortLen)

	copy(hsFrame[HSFrameHeaderLen:], loop.Local.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:], uint16(loop.Local.Port))

	copy(hsFrame[HSFrameHeaderLen+HSFramePKLen+HSFramePortLen:], loop.Remote.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+2*HSFramePKLen+HSFramePortLen:], uint16(loop.Remote.Port))

	return hsFrame
}

func NewHSFrameDMSGAccept(procID ProcID, loop routing.Loop) HSFrame {
	hsFrame := newHSFrame(procID, HSFrameTypeDMSGAccept, 2*HSFramePKLen+2*HSFramePortLen)

	copy(hsFrame[HSFrameHeaderLen:], loop.Local.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:], uint16(loop.Local.Port))

	copy(hsFrame[HSFrameHeaderLen+HSFramePKLen+HSFramePortLen:], loop.Remote.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+2*HSFramePKLen+HSFramePortLen:], uint16(loop.Remote.Port))

	return hsFrame
}

func NewHSFrameDMSGStopListening(procID ProcID, local routing.Addr) HSFrame {
	hsFrame := newHSFrame(procID, HSFrameTypeDMSGListen, HSFramePKLen+HSFramePortLen)

	copy(hsFrame[HSFrameHeaderLen:], local.PubKey[:])
	binary.BigEndian.PutUint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:], uint16(local.Port))

	return hsFrame
}

// ProcID gets ProcID from the HSFrame.
func (f HSFrame) ProcID() ProcID {
	return ProcID(binary.BigEndian.Uint16(f))
}

// SetProcID sets ProcID for the HSFrame.
func (f HSFrame) SetProcID(procID ProcID) {
	binary.BigEndian.PutUint16(f, uint16(procID))
}

// FrameType gets FrameType from the HSFrame.
func (f HSFrame) FrameType() HSFrameType {
	_ = f[HSFrameProcIDLen] // bounds check hint to compiler; see golang.org/issue/14808
	return HSFrameType(f[HSFrameProcIDLen])
}

// SetFrameType sets FrameType for the HSFrame.
func (f HSFrame) SetFrameType(frameType HSFrameType) {
	_ = f[HSFrameProcIDLen] // bounds check hint to compiler; see golang.org/issue/14808
	f[HSFrameProcIDLen] = byte(frameType)
}

func readHSFrame(r io.Reader) (HSFrame, error) {
	hsFrame := make(HSFrame, HSFrameHeaderLen)
	if _, err := io.ReadFull(r, hsFrame); err != nil {
		return nil, errors.Wrap(err, "error reading HS frame header")
	}

	hsFrame, err := readHSFrameBody(hsFrame, r)
	if err != nil {
		return nil, errors.Wrap(err, "error reading HS frame body")
	}

	return hsFrame, nil
}

func readHSFrameBody(hsFrame HSFrame, r io.Reader) (HSFrame, error) {
	switch hsFrame.FrameType() {
	case HSFrameTypeDMSGListen, HSFrameTypeDMSGListening:
		hsFrame = append(hsFrame, make([]byte, HSFramePKLen+HSFramePortLen)...)
	case HSFrameTypeDMSGDial, HSFrameTypeDMSGAccept:
		hsFrame = append(hsFrame, make([]byte, 2*HSFramePKLen+2*HSFramePortLen)...)
	}

	_, err := io.ReadFull(r, hsFrame[HSFrameHeaderLen:])
	return hsFrame, err
}
