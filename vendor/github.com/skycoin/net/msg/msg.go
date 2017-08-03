package msg

import (
	"encoding/binary"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
	"time"
)

type Message struct {
	Type uint8
	Seq  uint32
	Len  uint32
	Body []byte

	MessageStatus
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

type MessageStatus struct {
	Status int

	TransmittedAt time.Time
	AckedAt       time.Time
	Latency       time.Duration

	sync.RWMutex
}

func (ms *MessageStatus) Transmitted() {
	ms.Lock()
	ms.Status |= MSG_STATUS_TRANSMITTED
	ms.TransmittedAt = time.Now()
	ms.Unlock()
}

func (ms *MessageStatus) Acked() {
	ms.Lock()
	ms.Status |= MSG_STATUS_ACKED
	ms.AckedAt = time.Now()
	ms.Latency = ms.AckedAt.Sub(ms.TransmittedAt)
	ms.Unlock()
}
