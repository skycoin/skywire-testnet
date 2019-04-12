package appnet

import (
	"encoding/binary"
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
	case FailureFrame:
		return "FailureResp"
	case SuccessFrame:
		return "SuccessResp"
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

	// FailureFrame  represents frame type for failed requests.
	FailureFrame = 0xfe
	// SuccessFrame represents frame type for successful requests.
	SuccessFrame = 0xff
)

// Protocol implements full-duplex protocol for App to Node communication.
type Protocol struct {
	rw      io.ReadWriteCloser
	respMap *responseMap
}

// NewProtocol constructs a new Protocol.
func NewProtocol(rw io.ReadWriteCloser) *Protocol {
	return &Protocol{rw: rw, respMap: newResponseMap()}
}

// Call sends a frame of given type and awaits a response.
func (p *Protocol) Call(t FrameType, reqData []byte) ([]byte, error) {
	respID, respCh := p.respMap.add()
	if err := p.writeFrame(t, respID, reqData); err != nil {
		return nil, err
	}
	resp, ok := <-respCh
	if !ok {
		return nil, io.EOF
	}
	if resp.Type == FailureFrame {
		return nil, errors.New(string(resp.Data))
	}
	return resp.Data, nil
}

type (
	HandlerFunc func(p *Protocol, b []byte) ([]byte, error)
	HandlerMap  map[FrameType]HandlerFunc
)

func (p *Protocol) Serve(handlerMap HandlerMap) error {
	if handlerMap == nil {
		handlerMap = make(HandlerMap)
	}
	for {
		t, respID, payload, err := p.readFrame()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				return nil
			}
			return err
		}
		switch t {
		case SuccessFrame, FailureFrame:
			if respChan, ok := p.respMap.pull(respID); ok {
				respChan.send(t, payload)
			}
		default:
			handle, ok := handlerMap[t]
			if !ok {
				return fmt.Errorf("received unexpected frame of type: %s", t)
			}
			go func() {
				if resp, err := handle(p, payload); err != nil {
					_ = p.writeFrame(FailureFrame, respID, []byte(err.Error())) //nolint:errcheck
				} else {
					_ = p.writeFrame(SuccessFrame, respID, resp) //nolint:errcheck
				}
			}()
		}
	}
}

// Close closes underlying ReadWriter.
func (p *Protocol) Close() error {
	p.respMap.close()
	return p.rw.Close()
}

// a frame is formatted as follows:
// field: | size | type | id | payload |
// bytes: | 2    | 1    | 1  | ~       |
func (p *Protocol) writeFrame(t FrameType, respID responseID, payload []byte) error {
	f := make([]byte, 2+1+1+len(payload))
	binary.BigEndian.PutUint16(f[0:2], uint16(1+1+len(payload)))
	f[2] = byte(t)
	f[3] = byte(respID)
	copy(f[4:], payload)
	_, err := p.rw.Write(f)
	return err
}

func (p *Protocol) readFrame() (t FrameType, respID responseID, payload []byte, err error) {
	rawSize := make([]byte, 2)
	if _, err = io.ReadFull(p.rw, rawSize); err != nil {
		return
	}
	size := binary.BigEndian.Uint16(rawSize)
	frame := make([]byte, size)
	if _, err = io.ReadFull(p.rw, frame); err != nil {
		return
	}
	t = FrameType(frame[0])
	respID = responseID(frame[1])
	payload = frame[2:]
	return
}

type response struct {
	Type FrameType
	Data []byte
}

type (
	responseID   byte
	responseChan chan response
)

func (rc responseChan) send(t FrameType, p []byte) { rc <- response{Type: t, Data: p} }

type responseMap struct {
	sync.Mutex
	chMap map[responseID]responseChan
}

func newResponseMap() *responseMap {
	return &responseMap{chMap: make(map[responseID]responseChan)}
}

func (c *responseMap) add() (responseID, responseChan) {
	c.Lock()
	defer c.Unlock()

	ch := make(responseChan)
	for i := responseID(0); i < 255; i++ {
		if c.chMap[i] == nil {
			c.chMap[i] = ch
			return i, ch
		}
	}
	panic("appnet.Protocol: no free channels")
}

func (c *responseMap) pull(id responseID) (responseChan, bool) {
	c.Lock()
	ch, ok := c.chMap[id]
	delete(c.chMap, id)
	c.Unlock()
	return ch, ok
}

func (c *responseMap) close() {
	c.Lock()
	defer c.Unlock()

	for _, ch := range c.chMap {
		if ch == nil {
			continue
		}
		close(ch)
	}
	c.chMap = make(map[responseID]responseChan)
}
