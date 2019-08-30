package app2

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
)

const (
	HSFrameHeaderLength  = 5
	HSFrameProcIDLength  = 2
	HSFrameTypeLength    = 1
	HSFrameBodyLenLength = 2
	HSFrameMaxBodyLength = math.MaxUint16
)

var (
	// ErrHSFrameBodyTooLong is being returned when the body is too long to be
	// fit in the HSFrame
	ErrHSFrameBodyTooLong = errors.New("frame body is too long")
)

// HSFrameType identifies the type of a handshake frame.
type HSFrameType byte

const (
	HSFrameTypeDMSGListen HSFrameType = 10 + iota
	HSFrameTypeDMSGListening
	HSFrameTypeDMSGDial
	HSFrameTypeDMSGAccept
)

// HSFrame is the data unit for socket connection handshakes between Server and Client.
// It consists of header and body.
//
// Header is a big-endian encoded 5 bytes and is constructed as follows:
// | ProcID (2 bytes) | HSFrameType (1 byte) | BodyLen (2 bytes) |
//
// Body is a marshaled JSON structure
type HSFrame []byte

// NewHSFrame constructs new HSFrame.
func NewHSFrame(procID ProcID, frameType HSFrameType, body interface{}) (HSFrame, error) {
	bodyBytes, err := marshalHSFrameBody(body)
	if err != nil {
		return nil, err
	}

	hsFrame := make(HSFrame, HSFrameHeaderLength+len(bodyBytes))

	hsFrame.SetProcID(procID)
	hsFrame.SetFrameType(frameType)
	_ = hsFrame.SetBodyLen(len(bodyBytes))

	copy(hsFrame[HSFrameProcIDLength+HSFrameTypeLength+HSFrameBodyLenLength:], bodyBytes)

	return hsFrame, nil
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
	_ = f[HSFrameProcIDLength] // bounds check hint to compiler; see golang.org/issue/14808
	return HSFrameType(f[HSFrameProcIDLength])
}

// SetFrameType sets FrameType for the HSFrame.
func (f HSFrame) SetFrameType(frameType HSFrameType) {
	_ = f[HSFrameProcIDLength] // bounds check hint to compiler; see golang.org/issue/14808
	f[HSFrameProcIDLength] = byte(frameType)
}

// BodyLen gets BodyLen from the HSFrame.
func (f HSFrame) BodyLen() int {
	return int(binary.BigEndian.Uint16(f[HSFrameProcIDLength+HSFrameTypeLength:]))
}

// SetBodyLen sets BodyLen for the HSFrame.
func (f HSFrame) SetBodyLen(bodyLen int) error {
	if bodyLen > HSFrameMaxBodyLength {
		return ErrHSFrameBodyTooLong
	}

	binary.BigEndian.PutUint16(f[HSFrameProcIDLength+HSFrameTypeLength:], uint16(bodyLen))

	return nil
}

func marshalHSFrameBody(body interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if len(bodyBytes) > HSFrameMaxBodyLength {
		return nil, ErrHSFrameBodyTooLong
	}

	return bodyBytes, nil
}
