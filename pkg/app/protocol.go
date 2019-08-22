package app

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/skycoin/skycoin/src/util/logging"

	th "github.com/skycoin/skywire/internal/testhelpers"
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
		return "ConfirmLoop"
	case FrameSend:
		return "Send"
	case FrameClose:
		return "Close"
	case FrameFailure:
		return "Failure"
	case FrameSuccess:
		return "Success"
	}

	return fmt.Sprintf("Unknown(%d)", f)
}

const (
	// FrameInit represents Init frame type.
	FrameInit Frame = iota
	// FrameCreateLoop represents CreateLoop request frame type.
	FrameCreateLoop
	// FrameConfirmLoop represents ConfirmLoop request frame type.
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

// Logger is PackageLogger for app
var Logger = logging.MustGetLogger("Protocol")

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
	Logger.Debug(th.Trace("ENTER"))
	Logger.Debugf("%v cmd: %v, payload: %v\n", th.GetCaller(), cmd, payload)
	Logger.Debugf("%v CALLERS: %v\n", th.GetCaller(), th.GetCallers(3))

	id, resChan := p.chans.add()
	if err := p.writeFrame(cmd, id, payload); err != nil {
		Logger.Warnf("%v p.writeFrame(%v, %v, %v)  err: %v\n", th.GetCaller(), cmd, id, payload, err)

		return err
	}

	Logger.Debugf("%v waiting reply\n", th.GetCaller())
	frame, more := <-resChan
	if !more {
		return io.EOF
	}
	Logger.Debugf("%v received %v\n", th.GetCaller(), Frame(frame[0]))
	Logger.Debugf("Received %v\n", Payload{Frame(frame[0]), frame[2:]})

	if Frame(frame[0]) == FrameFailure {
		Logger.Warnf("%v writeFrame err: %v\n", th.GetCaller(), string(frame[2:]))
		return errors.New(string(frame[2:]))
	}

	if res == nil {
		return nil
	}

	return json.Unmarshal(frame[2:], res)
}

// Serve reads incoming frame, passes it to the handleFunc and writes results.
func (p *Protocol) Serve(handleFunc func(Frame, []byte) (interface{}, error)) error {
	Logger.Debug(th.Trace("ENTER"))
	defer Logger.Debug(th.Trace("EXIT"))
	var cntr uint64

	for {
		atomic.AddUint64(&cntr, 1)
		Logger.Debugf("%v CYCLE %03d START", th.GetCaller(), cntr)
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
	Logger.Debug(th.Trace("ENTER"))
	defer Logger.Debug(th.Trace("EXIT"))

	if p == nil {
		return nil
	}
	p.chans.closeAll()
	return p.rw.Close()
}

func (p *Protocol) writeFrame(frame Frame, id byte, payload interface{}) (err error) {
	Logger.Debug(th.Trace("ENTER"))
	defer Logger.Debug(th.Trace("EXIT"))

	var data []byte
	if err, ok := payload.(error); ok {
		data = []byte(err.Error())
	} else {
		data, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	Logger.WithField("payload", Payload{frame, data}).Info(th.GetCaller())

	packet := append([]byte{byte(frame), id}, data...)
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(packet)))
	_, err = p.rw.Write(append(buf, packet...))

	if err != nil {
		Logger.Warnf("% p.rw.Write err: %v\n", th.GetCaller(), err)
	}

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
