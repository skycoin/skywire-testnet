package messaging

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// MaxDataBodySize defines the maximum payload body size.
	MaxDataBodySize = 65518
)

var (
	// ErrDataTypeUnknown occurs when the data type is unknown.
	ErrDataTypeUnknown = errors.New("data type unknown and unregistered")

	// ErrDataTooLarge occurs when the data is too large.
	ErrDataTooLarge = errors.New("data too large")

	// ErrDataTooSmall occurs when the data is too small.
	ErrDataTooSmall = errors.New("data too small")

	// ErrHandshakeFailed occurs when a handshake fails.
	ErrHandshakeFailed = errors.New("handshake failed") // TODO(evanlinjin): Make this a struct.
)

// FrameType determines the type of payload.
type FrameType byte

func (f FrameType) String() string {
	switch f {
	case FrameTypeOpenChannel:
		return "OpenChannel"
	case FrameTypeChannelOpened:
		return "ChannelOpened"
	case FrameTypeCloseChannel:
		return "CloseChannel"
	case FrameTypeChannelClosed:
		return "ChannelClosed"
	case FrameTypeSend:
		return "Send"
	}

	return fmt.Sprintf("Unknown(%d)", f)
}

const (
	// FrameTypeOpenChannel defines frame for OpenChannel packet.
	FrameTypeOpenChannel FrameType = iota
	// FrameTypeChannelOpened defines frame for ChannelOpened packet.
	FrameTypeChannelOpened
	// FrameTypeCloseChannel defines frame for CloseChannel packet.
	FrameTypeCloseChannel
	// FrameTypeChannelClosed defines frame for ChannelClosed packet.
	FrameTypeChannelClosed
	// FrameTypeSend defines frame for Send packet.
	FrameTypeSend
)

/*
	<<< DATA >>>
*/

// Frame is structures as follows:
// - FrameType (1 byte)
// - FrameBody ([0,65518] bytes)
type Frame []byte

// MakeFrame makes a data with given type and body.
func MakeFrame(t FrameType, body []byte) Frame {
	p := make(Frame, len(body)+1)
	p[0] = byte(t)
	copy(p[1:], body)
	return p
}

// CheckSize checks the size of the data.
func (p Frame) CheckSize() error {
	if len(p) < 1 {
		return ErrDataTooSmall
	}
	if len(p) > MaxDataBodySize+1 {
		return ErrDataTooLarge
	}
	return nil
}

// Type returns the data type.
func (p Frame) Type() FrameType {
	return FrameType(p[0])
}

// Body returns the data body.
func (p Frame) Body() []byte {
	return p[1:]
}

/*
	<<< FRAME >>>
*/

// WriteFrame writes a frame to a given Writer.
// A frame is structured as follows:
// - FrameSize (2 bytes)
// - FrameType (1 byte)
// - ChannelID (1 byte)
// - Data ([3,65535] bytes)
func WriteFrame(w io.Writer, data Frame) (int, error) {
	if err := data.CheckSize(); err != nil {
		return 0, err
	}
	var (
		size   = uint16(len(data))
		packet = make([]byte, len(data)+2)
	)
	binary.BigEndian.PutUint16(packet[:2], size)
	copy(packet[2:], data)
	n, err := w.Write(packet)
	return n - 4, err
}

// ReadFrame read and decrypts data from a reader.
func ReadFrame(r io.Reader) (Frame, int, error) {
	// determine encrypted data size
	size, err := readUint16(r)
	if err != nil {
		return Frame{}, 0, err
	}
	if size > MaxDataBodySize {
		return Frame{}, 0, ErrDataTooLarge
	}
	// decrypt data
	plainText := make([]byte, size)
	n, err := io.ReadFull(r, plainText)
	if err != nil {
		return Frame{}, 0, err
	}

	// return results
	return Frame(plainText), n + 2, nil
}

func readUint16(r io.Reader) (uint16, error) {
	v := make([]byte, 2)
	if _, err := io.ReadFull(r, v); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(v), nil
}
