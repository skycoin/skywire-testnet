package app

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	bufSize = 32 * 1024
)

func init() {
	gob.Register(LoopAddr{})
	gob.Register(LoopMeta{})
	gob.Register(DataFrame{})
}

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

func (la *LoopAddr) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(la); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (la *LoopAddr) Decode(b []byte) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(la)
}

// LoopMeta stores addressing parameters of a loop packets.
type LoopMeta struct {
	Local  LoopAddr `json:"local"`
	Remote LoopAddr `json:"remote"`
}

func (l *LoopMeta) IsLoopback() bool {
	return l.Local.PubKey == l.Remote.PubKey
}

func (l LoopMeta) Swap() *LoopMeta {
	return &LoopMeta{Local: l.Remote, Remote: l.Local}
}

func (l *LoopMeta) String() string {
	return fmt.Sprintf("%s:%d|%s:%d", l.Local.PubKey, l.Local.Port, l.Remote.PubKey, l.Remote.Port)
}

func (l *LoopMeta) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(l); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (l *LoopMeta) Decode(b []byte) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(l)
}

// DataFrame represents message exchanged between App and Node.
type DataFrame struct {
	Meta LoopMeta `json:"meta"` // Can be remote or local Addr, depending on direction.
	Data []byte   `json:"data"` // Either ciphertext or plaintext (depending on usage).
}

func (df *DataFrame) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(df); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (df *DataFrame) Decode(b []byte) error {
	return gob.NewDecoder(bytes.NewReader(b)).Decode(df)
}

// LoopConn represents the App's perspective of a loop.
type LoopConn struct {
	net.Conn
	lm LoopMeta
}

func setAndServeLoop(lm LoopMeta) (*LoopConn, error) {
	conn, hostConn := net.Pipe()
	lc := &LoopConn{
		Conn: conn,
		lm:   lm,
	}
	if err := setLoopPipe(lc.lm, hostConn); err != nil {
		return nil, err
	}
	go func() {
		if err := lc.serve(hostConn); err != nil && err != io.ErrClosedPipe {
			log.Warnf("loop (%s) closed with error: %s", lc.lm.String(), err.Error())
		}
		rmLoopPipe(lc.lm)
	}()
	return lc, nil
}

func (lc *LoopConn) serve(hostConn net.Conn) error {
	for {
		buf := make([]byte, bufSize)
		n, err := hostConn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		frame := DataFrame{Meta: lc.lm, Data: buf[:n]}
		if _, err := _proto.Call(appnet.FrameData, frame.Encode()); err != nil {
			return err
		}
	}
}

func (lc *LoopConn) LocalAddr() net.Addr {
	return &lc.lm.Local
}

func (lc *LoopConn) RemoteAddr() net.Addr {
	return &lc.lm.Remote
}
