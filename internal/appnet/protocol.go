package appnet

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// FrameType defines type for all App frames.
type FrameType byte

func (f FrameType) String() string {
	switch f {
	case FrameCreateLoop:
		return "CreateLoop"
	case FrameConfirmLoop:
		return "ConfirmLoop"
	case FrameData:
		return "Data"
	case FrameCloseLoop:
		return "Close"
	default:
		return fmt.Sprintf("Unknown(%d)", f)
	}
}

const (
	// FrameCreateLoop represents CreateLoop request frame type.
	FrameCreateLoop FrameType = iota
	// FrameConfirmLoop represents ConfirmLoop request frame type.
	FrameConfirmLoop
	// FrameData represents Send frame type.
	FrameData
	// FrameCloseLoop represents Close frame type
	FrameCloseLoop

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
	return &Protocol{rw, &chanList{chMap: map[byte]chan []byte{}}}
}

// Send sends command FrameType with payload and awaits for response.
func (p *Protocol) Send(cmd FrameType, payload, res interface{}) error {
	id, resChan := p.chans.add()
	if err := p.writeFrame(cmd, id, payload); err != nil {
		return err
	}

	frame, more := <-resChan
	if !more {
		return io.EOF
	}

	if FrameType(frame[0]) == FrameFailure {
		return errors.New(string(frame[2:]))
	}

	if res == nil {
		return nil
	}

	return json.Unmarshal(frame[2:], res)
}

// Serve reads incoming frame, passes it to the handleFunc and writes results.
func (p *Protocol) Serve(handleFunc func(FrameType, []byte) (interface{}, error)) error {
	for {
		frame, err := p.readFrame()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				return nil
			}

			return err
		}

		fType := FrameType(frame[0])
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
				p.writeFrame(FrameSuccess, id, nil) // nolint: errcheck
				return
			}

			res, err := handleFunc(fType, frame[2:])
			if err != nil {
				p.writeFrame(FrameFailure, id, err) // nolint: errcheck
				return
			}

			p.writeFrame(FrameSuccess, id, res) // nolint: errcheck
		}()
	}
}

// Close closes underlying ReadWriter.
func (p *Protocol) Close() error {
	p.chans.closeAll()
	return p.rw.Close()
}

func (p *Protocol) writeFrame(frame FrameType, id byte, payload interface{}) (err error) {
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
	chMap map[byte]chan []byte
}

func (c *chanList) add() (byte, chan []byte) {
	c.Lock()
	defer c.Unlock()

	ch := make(chan []byte)
	for i := byte(0); i < 255; i++ {
		if c.chMap[i] == nil {
			c.chMap[i] = ch
			return i, ch
		}
	}

	panic("no free channels")
}

func (c *chanList) pull(id byte) chan []byte {
	c.Lock()
	ch := c.chMap[id]
	delete(c.chMap, id)
	c.Unlock()

	return ch
}

func (c *chanList) closeAll() {
	c.Lock()
	defer c.Unlock()

	for _, ch := range c.chMap {
		if ch == nil {
			continue
		}

		close(ch)
	}

	c.chMap = make(map[byte]chan []byte)
}
