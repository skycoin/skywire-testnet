package app

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/internal/ioutil"
	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	bufSize = 32 * 1024
)

// LoopAddr implements net.Conn for connections between apps and node.
type LoopAddr struct {
	PubKey cipher.PubKey `json:"pk"`
	Port   uint16        `json:"port"`
}

// Network returns custom skywire Network type.
func (la *LoopAddr) Network() string {
	return "skywire"
}

// String implements fmt.Stringer
func (la *LoopAddr) String() string {
	return fmt.Sprintf("%s:%d", la.PubKey, la.Port)
}

// LoopMeta stores addressing parameters of a loop packets.
type LoopMeta struct {
	Local  LoopAddr `json:"local"`
	Remote LoopAddr `json:"remote"`
}

func (l *LoopMeta) String() string {
	return fmt.Sprintf("%s:%d <-> %s:%d", l.Local.PubKey, l.Local.Port, l.Remote.PubKey, l.Remote.Port)
}

// DataFrame represents message exchanged between App and Node.
type DataFrame struct {
	Meta LoopMeta `json:"meta"` // Can be remote or local Addr, depending on direction.
	Data []byte   `json:"data"`
}

// LoopConn represents the App's perspective of a loop.
type LoopConn struct {
	net.Conn
	rw *ioutil.AckReadWriter
	lm LoopMeta
}

func newLoopConn(lm LoopMeta) *LoopConn {
	conn, hostConn := net.Pipe()
	lc := &LoopConn{
		Conn: conn,
		rw:   ioutil.NewAckReadWriter(conn, 100*time.Millisecond),
		lm:   lm,
	}

	lc.serve(hostConn)
	return lc
}

func (lc *LoopConn) serve(hostConn net.Conn) {
	runLoop := func() error {
		for {
			buf := make([]byte, bufSize)
			n, err := hostConn.Read(buf)
			if err != nil {
				if err == io.ErrClosedPipe {
					return nil
				}
				return err
			}
			frame := DataFrame{Meta: lc.lm, Data: buf[:n]}
			if err := _proto.Send(appnet.FrameData, frame, nil); err != nil {
				return err
			}
		}
	}

	_mu.Lock()
	_loops[lc.lm] = hostConn
	_mu.Unlock()

	go func() {
		if err := runLoop(); err != nil {
			log.Warnf("loop (%s) closed with error: %s", lc.lm, err.Error())
		}
	}()

	_mu.Lock()
	if _, ok := _loops[lc.lm]; !ok {
		_ = _proto.Send(appnet.FrameCloseLoop, lc.lm, nil) //nolint:errcheck
	}
	delete(_loops, lc.lm)
	_mu.Unlock()
}

func (lc *LoopConn) Read(p []byte) (n int, err error) {
	return lc.rw.Read(p)
}

func (lc *LoopConn) Write(p []byte) (n int, err error) {
	return lc.rw.Write(p)
}

func (lc *LoopConn) Close() error {
	return lc.rw.Close()
}

func (lc *LoopConn) LocalAddr() net.Addr {
	return &lc.lm.Local
}

func (lc *LoopConn) RemoteAddr() net.Addr {
	return &lc.lm.Remote
}
