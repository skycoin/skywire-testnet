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
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if len(bodyBytes) > HSFrameMaxBodyLength {
		return nil, ErrHSFrameBodyTooLong
	}

	hsFrame := make(HSFrame, HSFrameHeaderLength+len(bodyBytes))

	binary.BigEndian.PutUint16(hsFrame, uint16(procID))
	hsFrame[HSFrameProcIDLength] = byte(frameType)
	binary.BigEndian.PutUint16(hsFrame[HSFrameProcIDLength+HSFrameTypeLength:], uint16(len(bodyBytes)))

	copy(hsFrame[HSFrameProcIDLength+HSFrameTypeLength+HSFrameBodyLenLength:], bodyBytes)

	return hsFrame, nil
}
