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
	rw     io.ReadWriteCloser
	waiter *responseWaiter
}

// NewProtocol constructs a new Protocol.
func NewProtocol(rw io.ReadWriteCloser) *Protocol {
	return &Protocol{rw: rw, waiter: newResponseWaiter()}
}

// Call sends a frame of given type and awaits a response.
func (p *Protocol) Call(t FrameType, reqData []byte) ([]byte, error) {
	waitID, waitCh := p.waiter.add()
	if err := p.writeFrame(t, waitID, reqData); err != nil {
		return nil, err
	}
	//fmt.Printf(">> Proto.Call: t(%s) id(%d) req(%v)\n", t, waitID, reqData)
	resp, ok := <-waitCh
	if !ok {
		return nil, io.EOF
	}
	if resp.Type == FailureFrame {
		return nil, errors.New(string(resp.Data))
	}
	//fmt.Printf("<< Proto.Call: t(%s) \n", t)
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
		//fmt.Printf("\tProto.Serve: t(%s) id(%d) len(%d)\n", t, respID, len(payload))
		switch t {
		case SuccessFrame, FailureFrame:
			if waitCh, ok := p.waiter.pull(respID); ok {
				go waitCh.pushResponse(t, payload)
			}
		default:
			handle, ok := handlerMap[t]
			if !ok {
				handle = func(*Protocol, []byte) ([]byte, error) {
					return nil, fmt.Errorf("received unexpected frame of type: %s", t)
				}
			}
			go func(handle HandlerFunc, payload []byte) {
				var (
					respType    FrameType
					respPayload []byte
				)
				if resp, err := handle(p, payload); err != nil {
					respType = FailureFrame
					respPayload = []byte(err.Error())
				} else {
					respType = SuccessFrame
					respPayload = resp
				}
				if err := p.writeFrame(respType, respID, respPayload); err != nil {
					fmt.Println("\tPROTO: failed to write response:", err.Error())
				}
				fmt.Println("\tPROTO: responded")
			}(handle, payload)
		}
	}
}

// Close closes underlying ReadWriter.
func (p *Protocol) Close() error {
	p.waiter.close()
	return p.rw.Close()
}

// a frame is formatted as follows:
// field: | size | type | id | payload |
// bytes: | 2    | 1    | 1  | ~       |
func (p *Protocol) writeFrame(t FrameType, respID waitID, payload []byte) error {
	f := make([]byte, 2+1+1+len(payload))
	binary.BigEndian.PutUint16(f[0:2], uint16(1+1+len(payload)))
	f[2] = byte(t)
	f[3] = byte(respID)
	copy(f[4:], payload)
	_, err := p.rw.Write(f)
	return err
}

func (p *Protocol) readFrame() (t FrameType, respID waitID, payload []byte, err error) {
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
	respID = waitID(frame[1])
	payload = frame[2:]
	return
}

type response struct {
	Type FrameType
	Data []byte
}

type (
	waitID   byte
	waitChan chan response
)

func (rc waitChan) pushResponse(t FrameType, p []byte) { rc <- response{Type: t, Data: p} }

type responseWaiter struct {
	sync.Mutex
	waiters [256]waitChan
	i       uint8
}

func newResponseWaiter() *responseWaiter {
	return new(responseWaiter)
}

func (c *responseWaiter) add() (waitID, waitChan) {
	c.Lock()
	i, ch := c.i, make(waitChan)
	c.i, c.waiters[i] = c.i+1, ch
	c.Unlock()
	return waitID(i), ch
}

func (c *responseWaiter) pull(id waitID) (waitChan, bool) {
	c.Lock()
	ch := c.waiters[id]
	c.waiters[id] = nil
	c.Unlock()
	return ch, ch != nil
}

func (c *responseWaiter) close() {
	c.Lock()
	for _, ch := range c.waiters {
		if ch != nil {
			close(ch)
		}
	}
	c.Unlock()
}
