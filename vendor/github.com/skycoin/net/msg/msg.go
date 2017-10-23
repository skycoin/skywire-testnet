package msg

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

type Interface interface {
	Bytes() []byte
	TotalSize() int
	Transmitted()
	Acked()
	GetRTT() time.Duration
}

type Message struct {
	Type uint8
	Seq  uint32
	Len  uint32
	Body []byte

	sync.RWMutex

	Status        int
	TransmittedAt time.Time
	AckedAt       time.Time
	rtt           time.Duration
}

func NewByHeader(header []byte) *Message {
	m := &Message{}
	m.Type = uint8(header[0])
	m.Seq = binary.BigEndian.Uint32(header[MSG_SEQ_BEGIN:MSG_SEQ_END])
	m.Len = binary.BigEndian.Uint32(header[MSG_LEN_BEGIN:MSG_LEN_END])
	if m.Len > MAX_MESSAGE_SIZE {
		panic(fmt.Errorf("msg len(%d) >  max len(%d)", m.Len, MAX_MESSAGE_SIZE))
	}

	m.Body = make([]byte, m.Len)

	return m
}

func New(t uint8, seq uint32, bytes []byte) *Message {
	return &Message{Type: t, Seq: seq, Len: uint32(len(bytes)), Body: bytes}
}

func (msg *Message) String() string {
	return fmt.Sprintf("Msg Type:%d, Seq:%d, Len:%d, Body:%x", msg.Type, msg.Seq, msg.Len, msg.Body)
}

func (msg *Message) GetHashId() cipher.SHA256 {
	return cipher.SumSHA256(msg.Body)
}

func (msg *Message) Bytes() []byte {
	result := make([]byte, MSG_HEADER_SIZE+msg.Len)
	result[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(result[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.Seq)
	binary.BigEndian.PutUint32(result[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	copy(result[MSG_HEADER_END:], msg.Body)
	return result
}

func (msg *Message) PkgBytes() []byte {
	result := make([]byte, PKG_HEADER_SIZE+MSG_HEADER_SIZE+msg.Len)
	m := result[PKG_HEADER_SIZE:]
	m[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(m[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.Seq)
	binary.BigEndian.PutUint32(m[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	copy(m[MSG_HEADER_END:], msg.Body)
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(result[PKG_CRC32_BEGIN:], checksum)
	return result
}

func (msg *Message) HeaderBytes() []byte {
	result := make([]byte, MSG_HEADER_SIZE)
	result[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(result[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.Seq)
	binary.BigEndian.PutUint32(result[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	return result
}

func (msg *Message) TotalSize() int {
	return MSG_HEADER_SIZE + len(msg.Body)
}

func (msg *Message) Transmitted() {
	msg.Lock()
	msg.Status |= MSG_STATUS_TRANSMITTED
	msg.TransmittedAt = time.Now()
	msg.Unlock()
}

func (msg *Message) Acked() {
	msg.Lock()
	msg.Status |= MSG_STATUS_ACKED
	msg.AckedAt = time.Now()
	msg.rtt = msg.AckedAt.Sub(msg.TransmittedAt)
	msg.Unlock()
}

func (msg *Message) GetRTT() (rtt time.Duration) {
	msg.RLock()
	rtt = msg.rtt
	msg.RUnlock()
	return
}

type UDPMessage struct {
	*Message

	miss        uint32
	resendTimer *time.Timer
}

func NewUDP(t uint8, seq uint32, bytes []byte) *UDPMessage {
	return &UDPMessage{
		Message: New(t, seq, bytes),
	}
}

func (msg *UDPMessage) Transmitted() {
	msg.Lock()
	msg.Status |= MSG_STATUS_TRANSMITTED
	msg.TransmittedAt = time.Now()
	msg.Unlock()
}

func (msg *UDPMessage) SetRTO(rto time.Duration, fn func() error) {
	msg.Lock()
	msg.setRTO(rto, fn)
	msg.Unlock()
}

func (msg *UDPMessage) setRTO(rto time.Duration, fn func() error) {
	msg.resendTimer = time.AfterFunc(rto, func() {
		msg.Lock()
		if msg.Status&MSG_STATUS_ACKED > 0 {
			msg.Unlock()
			return
		}
		msg.ResetMiss()
		err := fn()
		if err == nil {
			msg.setRTO(rto, fn)
		}
		msg.Unlock()
	})
}

func (msg *UDPMessage) Acked() {
	msg.Lock()
	msg.Status |= MSG_STATUS_ACKED
	msg.AckedAt = time.Now()
	msg.rtt = msg.AckedAt.Sub(msg.TransmittedAt)
	msg.resendTimer.Stop()
	msg.Unlock()
}

func (msg *UDPMessage) Miss() uint32 {
	return atomic.AddUint32(&msg.miss, 1)
}

func (msg *UDPMessage) ResetMiss() {
	atomic.StoreUint32(&msg.miss, 0)
}
