package app

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Frame defines type for all App frames.
type Frame byte

func (f Frame) String() string {
	switch f {
	case FrameInit:
		return "Init"
	case FrameCreateLoop:
		return "CreateLoop"
	case FrameConfirmLoop:
		return "OnConfirmLoop"
	case FrameSend:
		return "Send"
	case FrameClose:
		return "Close"
	}

	return fmt.Sprintf("Unknown(%d)", f)
}

const (
	// FrameInit represents Init frame type.
	FrameInit Frame = iota
	// FrameCreateLoop represents CreateLoop request frame type.
	FrameCreateLoop
	// FrameConfirmLoop represents OnConfirmLoop request frame type.
	FrameConfirmLoop
	// FrameSend represents Send frame type.
	FrameSend
	// FrameClose represents Close frame type
	FrameClose

	// FrameFailure  represents frame type for failed requests.
	FrameFailure = 0xfe
	// FrameSuccess represents frame type for successful requests.
	FrameSuccess = 0xff
)

// Protocol implements full-duplex protocol for App to Node communication.
type Protocol struct {
	rw    io.ReadWriteCloser
	chans *chanList
}

// NewProtocol constructs a new Protocol.
func NewProtocol(rw io.ReadWriteCloser) *Protocol {
	return &Protocol{rw, &chanList{chans: map[byte]chan []byte{}}}
}

// Send sends command Frame with payload and awaits for response.
func (p *Protocol) Send(cmd Frame, payload, res interface{}) error {
	id, resChan := p.chans.add()
	if err := p.writeFrame(cmd, id, payload); err != nil {
		return err
	}

	frame, more := <-resChan
	if !more {
		return io.EOF
	}

	if Frame(frame[0]) == FrameFailure {
		return errors.New(string(frame[2:]))
	}

	if res == nil {
		return nil
	}

	return json.Unmarshal(frame[2:], res)
}

// Serve reads incoming frame, passes it to the handleFunc and writes results.
func (p *Protocol) Serve(handleFunc func(Frame, []byte) (interface{}, error)) error {
	for {
		frame, err := p.readFrame()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				return nil
			}

			return err
		}

		fType := Frame(frame[0])
		id := frame[1]

		var resChan chan []byte
		if fType == FrameFailure || fType == FrameSuccess {
			resChan = p.chans.pull(id)
			if resChan == nil {
				continue
			}
			resChan <- frame
			continue
		}

		go func() {
			if handleFunc == nil {
				if err := p.writeFrame(FrameSuccess, id, nil); err != nil {
					log.WithError(err).Warn("Failed to write frame")
				}
				return
			}

			res, err := handleFunc(fType, frame[2:])
			if err != nil {
				if err := p.writeFrame(FrameFailure, id, err); err != nil {
					log.WithError(err).Warn("Failed to write frame")
				}
				return
			}

			if err := p.writeFrame(FrameSuccess, id, res); err != nil {
				log.WithError(err).Warn("Failed to write frame")
			}
		}()
	}
}

// Close closes underlying ReadWriter.
func (p *Protocol) Close() error {
	if p == nil {
		return nil
	}
	p.chans.closeAll()
	return p.rw.Close()
}

func (p *Protocol) writeFrame(frame Frame, id byte, payload interface{}) (err error) {
	var data []byte
	if err, ok := payload.(error); ok {
		data = []byte(err.Error())
	} else {
		data, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	packet := append([]byte{byte(frame), id}, data...)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(packet)))
	_, err = p.rw.Write(append(buf, packet...))
	return err
}

func (p *Protocol) readFrame() (frame []byte, err error) {
	size := make([]byte, 2)
	if _, err = io.ReadFull(p.rw, size); err != nil {
		return
	}

	frame = make([]byte, binary.BigEndian.Uint16(size))
	if _, err = io.ReadFull(p.rw, frame); err != nil {
		return
	}

	return frame, nil
}

type chanList struct {
	sync.Mutex

	chans map[byte]chan []byte
}

func (c *chanList) add() (byte, chan []byte) {
	c.Lock()
	defer c.Unlock()

	ch := make(chan []byte)
	for i := byte(0); i < 255; i++ {
		if c.chans[i] == nil {
			c.chans[i] = ch
			return i, ch
		}
	}

	panic("no free channels")
}

func (c *chanList) pull(id byte) chan []byte {
	c.Lock()
	ch := c.chans[id]
	delete(c.chans, id)
	c.Unlock()

	return ch
}

func (c *chanList) closeAll() {
	c.Lock()
	defer c.Unlock()

	for _, ch := range c.chans {
		if ch == nil {
			continue
		}

		close(ch)
	}

	c.chans = make(map[byte]chan []byte)
}
