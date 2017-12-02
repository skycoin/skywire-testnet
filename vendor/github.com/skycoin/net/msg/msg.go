package msg

import (
	"encoding/binary"
	"fmt"
	"github.com/google/btree"
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
	PkgBytes() []byte
}

type Message struct {
	Type uint8
	seq  uint32
	Len  uint32
	Body []byte

	sync.RWMutex

	status        int
	transmittedAt time.Time
	ackedAt       time.Time
	rtt           time.Duration

	cache []byte
}

func NewByHeader(header []byte) *Message {
	m := &Message{}
	m.Type = uint8(header[0])
	m.seq = binary.BigEndian.Uint32(header[MSG_SEQ_BEGIN:MSG_SEQ_END])
	m.Len = binary.BigEndian.Uint32(header[MSG_LEN_BEGIN:MSG_LEN_END])
	if m.Len > MAX_MESSAGE_SIZE {
		panic(fmt.Errorf("msg len(%d) >  max len(%d)", m.Len, MAX_MESSAGE_SIZE))
	}

	m.Body = make([]byte, m.Len)

	return m
}

func New(t uint8, seq uint32, bytes []byte) *Message {
	return &Message{Type: t, seq: seq, Len: uint32(len(bytes)), Body: bytes}
}

func NewWithoutSeq(t uint8, bytes []byte) *Message {
	return &Message{Type: t, Len: uint32(len(bytes)), Body: bytes}
}

func (msg *Message) String() string {
	return fmt.Sprintf("Msg Type:%d, Seq:%d, Len:%d, Status:%x", msg.Type, msg.GetSeq(), msg.Len, msg.Status())
}

func (msg *Message) Status() (s int) {
	msg.RLock()
	s = msg.status
	msg.RUnlock()
	return
}

func (msg *Message) SetSeq(seq uint32) {
	atomic.StoreUint32(&msg.seq, seq)
}

func (msg *Message) GetSeq() uint32 {
	return atomic.LoadUint32(&msg.seq)
}

func (msg *Message) GetHashId() cipher.SHA256 {
	return cipher.SumSHA256(msg.Body)
}

func (msg *Message) Bytes() (result []byte) {
	msg.RLock()
	result = msg.cache
	msg.RUnlock()
	if len(result) > 0 {
		return
	}

	result = make([]byte, MSG_HEADER_SIZE+msg.Len)
	result[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(result[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.GetSeq())
	binary.BigEndian.PutUint32(result[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	copy(result[MSG_HEADER_END:], msg.Body)
	msg.Lock()
	msg.cache = result
	msg.Unlock()
	return result
}

func (msg *Message) PkgBytes() (result []byte) {
	msg.RLock()
	result = msg.cache
	msg.RUnlock()
	if len(result) > 0 {
		return
	}

	result = make([]byte, PKG_HEADER_SIZE+MSG_HEADER_SIZE+msg.Len)
	m := result[PKG_HEADER_SIZE:]
	m[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(m[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.GetSeq())
	binary.BigEndian.PutUint32(m[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	copy(m[MSG_HEADER_END:], msg.Body)
	checksum := crc32.ChecksumIEEE(m)
	binary.BigEndian.PutUint32(result[PKG_CRC32_BEGIN:], checksum)
	msg.Lock()
	msg.cache = result
	msg.Unlock()
	return
}

func (msg *Message) PkgBytesLen() int {
	return int(PKG_HEADER_SIZE + MSG_HEADER_SIZE + msg.Len)
}

func (msg *Message) HeaderBytes() []byte {
	result := make([]byte, MSG_HEADER_SIZE)
	result[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(result[MSG_SEQ_BEGIN:MSG_SEQ_END], msg.GetSeq())
	binary.BigEndian.PutUint32(result[MSG_LEN_BEGIN:MSG_LEN_END], msg.Len)
	return result
}

func (msg *Message) TotalSize() int {
	msg.RLock()
	defer msg.RUnlock()
	if len(msg.cache) > 0 {
		return len(msg.cache)
	}
	return MSG_HEADER_SIZE + len(msg.Body)
}

func (msg *Message) Transmitted() {
	msg.Lock()
	msg.status |= MSG_STATUS_TRANSMITTED
	msg.transmittedAt = time.Now()
	msg.Unlock()
}

func (msg *Message) Acked() {
	msg.Lock()
	msg.status |= MSG_STATUS_ACKED
	msg.ackedAt = time.Now()
	msg.rtt = msg.ackedAt.Sub(msg.transmittedAt)
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
	resendCnt   uint32
	resendTimer *time.Timer

	delivered     uint64
	deliveredTime time.Time
	sentTime      time.Time
	isAppLimited  int
}

func NewUDP(t uint8, seq uint32, bytes []byte) *UDPMessage {
	return &UDPMessage{
		Message: New(t, seq, bytes),
	}
}

func NewUDPWithoutSeq(t uint8, bytes []byte) *UDPMessage {
	return &UDPMessage{
		Message: NewWithoutSeq(t, bytes),
	}
}

func (msg *UDPMessage) Transmitted() {
	msg.Lock()
	msg.status |= MSG_STATUS_TRANSMITTED
	msg.transmittedAt = time.Now()
	msg.Unlock()
}

func (msg *UDPMessage) UpdateState(delivered uint64, deliveredTime, sentTime time.Time, isAppLimited int) {
	msg.Lock()
	msg.delivered = delivered
	msg.deliveredTime = deliveredTime
	msg.sentTime = sentTime
	msg.isAppLimited = isAppLimited
	msg.Unlock()
}

func (msg *UDPMessage) SetRTO(rto time.Duration, fn func() error) {
	msg.Lock()
	msg.resendTimer = time.AfterFunc(rto*time.Duration((msg.resendCnt)*3/2+1), func() {
		msg.Lock()
		if msg.status&MSG_STATUS_ACKED > 0 {
			msg.Unlock()
			return
		}
		msg.resendCnt++
		msg.Unlock()
		msg.ResetMiss()
		err := fn()
		if err == nil {
			msg.SetRTO(rto, fn)
		}
	})
	msg.Unlock()
}

func (msg *UDPMessage) Acked() {
	msg.Lock()
	msg.status |= MSG_STATUS_ACKED
	msg.ackedAt = time.Now()
	msg.rtt = msg.ackedAt.Sub(msg.transmittedAt)
	if msg.resendTimer != nil {
		msg.resendTimer.Stop()
	}
	msg.Unlock()
}

func (msg *UDPMessage) Miss() uint32 {
	return atomic.LoadUint32(&msg.miss)
}

func (msg *UDPMessage) AddMiss() uint32 {
	return atomic.AddUint32(&msg.miss, 1)
}

func (msg *UDPMessage) ResetMiss() {
	atomic.StoreUint32(&msg.miss, 0)
}

func (msg *UDPMessage) GetDelivered() uint64 {
	return msg.delivered
}

func (msg *UDPMessage) GetDeliveredTime() time.Time {
	return msg.deliveredTime
}

func (msg *UDPMessage) GetSentTime() time.Time {
	return msg.sentTime
}

func (msg *UDPMessage) GetTransmittedTime() time.Time {
	return msg.transmittedAt
}

func (msg *UDPMessage) Less(b btree.Item) bool {
	return msg.GetSeq() < b.(*UDPMessage).GetSeq()
}
