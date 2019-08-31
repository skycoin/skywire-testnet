package app2

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
)

const (
	HSFrameHeaderLen  = 5
	HSFrameProcIDLen  = 2
	HSFrameTypeLen    = 1
	HSFrameBodyLenLen = 2
	HSFrameMaxBodyLen = math.MaxUint16
)

var (
	// ErrHSFrameBodyTooLarge is being returned when the body is too long to be
	// fit in the HSFrame
	ErrHSFrameBodyTooLarge = errors.New("frame body is too long")
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

	hsFrame := make(HSFrame, HSFrameHeaderLen+len(bodyBytes))

	hsFrame.SetProcID(procID)
	hsFrame.SetFrameType(frameType)
	_ = hsFrame.SetBodyLen(len(bodyBytes))

	copy(hsFrame[HSFrameProcIDLen+HSFrameTypeLen+HSFrameBodyLenLen:], bodyBytes)

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
	_ = f[HSFrameProcIDLen] // bounds check hint to compiler; see golang.org/issue/14808
	return HSFrameType(f[HSFrameProcIDLen])
}

// SetFrameType sets FrameType for the HSFrame.
func (f HSFrame) SetFrameType(frameType HSFrameType) {
	_ = f[HSFrameProcIDLen] // bounds check hint to compiler; see golang.org/issue/14808
	f[HSFrameProcIDLen] = byte(frameType)
}

// BodyLen gets BodyLen from the HSFrame.
func (f HSFrame) BodyLen() int {
	return int(binary.BigEndian.Uint16(f[HSFrameProcIDLen+HSFrameTypeLen:]))
}

// SetBodyLen sets BodyLen for the HSFrame.
func (f HSFrame) SetBodyLen(bodyLen int) error {
	if bodyLen > HSFrameMaxBodyLen {
		return ErrHSFrameBodyTooLarge
	}

	binary.BigEndian.PutUint16(f[HSFrameProcIDLen+HSFrameTypeLen:], uint16(bodyLen))

	return nil
}

func marshalHSFrameBody(body interface{}) ([]byte, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if len(bodyBytes) > HSFrameMaxBodyLen {
		return nil, ErrHSFrameBodyTooLarge
	}

	return bodyBytes, nil
}
