package conn

import (
	"container/list"
	"net"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	ctxId uint32
)

type Connection interface {
	ReadLoop() error
	WriteLoop() error
	Write(bytes []byte) error
	GetChanIn() <-chan []byte
	GetChanOut() chan<- []byte
	Close()
	IsClosed() bool

	GetContextLogger() *log.Entry
	SetContextLogger(*log.Entry)

	GetRemoteAddr() net.Addr
	IsTCP() bool
	IsUDP() bool

	// Get last time about read bytes from connection
	GetLastTime() int64
	// Get sent bytes count
	GetSentBytes() uint64
	// Get received bytes count
	GetReceivedBytes() uint64

	NewPendingChannel() (channel int)
	DeletePendingChannel(channel int)
	WriteToChannel(channel int, bytes []byte) (err error)

	WaitForDisconnected()

	WriteSyn(bytes []byte) (err error)

	SetCrypto(crypto *Crypto)
	GetCrypto() *Crypto

	SetStatusToError(err error)
}

type ConnCommonFields struct {
	seq uint32 // id of last message, increment every new message

	HighestACKedSequenceNumber uint32 // highest packet that has been ACKed
	LastAck                    int64  // last time an ACK of receipt was received (better to store id of highest packet id with an ACK?)

	lastReadTime int64

	sentBytes     uint64
	receivedBytes uint64

	Status int // STATUS_CONNECTING, STATUS_CONNECTED, STATUS_ERROR
	err    error

	In           chan []byte
	Out          chan []byte
	closed       bool
	FieldsMutex  sync.RWMutex
	WriteMutex   sync.Mutex
	disconnected chan struct{}

	ctxLogger atomic.Value

	crypto      atomic.Value
	cryptoMutex sync.Mutex
	cryptoCond  *sync.Cond

	directlyHistory      *list.List
	directlyHistoryMutex sync.Mutex
}

func NewConnCommonFileds() *ConnCommonFields {
	entry := log.WithField("ctxId", atomic.AddUint32(&ctxId, 1))
	fields := &ConnCommonFields{
		lastReadTime:    time.Now().Unix(),
		In:              make(chan []byte, 128),
		Out:             make(chan []byte, 1),
		disconnected:    make(chan struct{}),
		directlyHistory: list.New(),
	}
	fields.cryptoCond = sync.NewCond(&fields.cryptoMutex)
	fields.ctxLogger.Store(entry)
	return fields
}

func (c *ConnCommonFields) SetStatusToConnected() {
	c.FieldsMutex.Lock()
	c.Status = STATUS_CONNECTED
	c.FieldsMutex.Unlock()
}

func (c *ConnCommonFields) SetStatusToError(err error) {
	c.FieldsMutex.Lock()
	if c.Status == STATUS_ERROR {
		c.FieldsMutex.Unlock()
		return
	}
	c.Status = STATUS_ERROR
	c.err = err
	c.FieldsMutex.Unlock()
	c.GetContextLogger().Debugf("SetStatusToError %v", err)
}

func (c *ConnCommonFields) GetStatusError() (err error) {
	c.FieldsMutex.RLock()
	if c.Status != STATUS_ERROR {
		c.FieldsMutex.RUnlock()
		return
	}
	err = c.err
	c.FieldsMutex.RUnlock()
	return
}

func (c *ConnCommonFields) UpdateLastAck(s uint32) {
	c.FieldsMutex.Lock()
	c.LastAck = time.Now().Unix()
	if s > c.HighestACKedSequenceNumber {
		c.HighestACKedSequenceNumber = s
	}
	c.FieldsMutex.Unlock()
}

func (c *ConnCommonFields) GetContextLogger() *log.Entry {
	return c.ctxLogger.Load().(*log.Entry)
}

func (c *ConnCommonFields) SetContextLogger(l *log.Entry) {
	c.ctxLogger.Store(l)
}

func (c *ConnCommonFields) GetChanOut() chan<- []byte {
	return c.Out
}

func (c *ConnCommonFields) GetChanIn() <-chan []byte {
	return c.In
}

func (c *ConnCommonFields) Close() {
	c.FieldsMutex.Lock()
	defer c.FieldsMutex.Unlock()

	if c.closed {
		return
	}
	c.closed = true

	c.cryptoCond.Broadcast()

	close(c.In)
	close(c.Out)
	close(c.disconnected)
}

func (c *ConnCommonFields) IsClosed() bool {
	c.FieldsMutex.RLock()
	defer c.FieldsMutex.RUnlock()
	return c.closed
}

func (c *ConnCommonFields) WaitForDisconnected() {
	<-c.disconnected
}

func (c *ConnCommonFields) GetLastTime() int64 {
	return atomic.LoadInt64(&c.lastReadTime)
}

func (c *ConnCommonFields) UpdateLastTime() {
	atomic.StoreInt64(&c.lastReadTime, time.Now().Unix())
}

func (c *ConnCommonFields) GetSentBytes() uint64 {
	return atomic.LoadUint64(&c.sentBytes)
}

func (c *ConnCommonFields) AddSentBytes(n int) {
	atomic.AddUint64(&c.sentBytes, uint64(n))
}

func (c *ConnCommonFields) GetReceivedBytes() uint64 {
	return atomic.LoadUint64(&c.receivedBytes)
}

func (c *ConnCommonFields) AddReceivedBytes(n int) {
	atomic.AddUint64(&c.receivedBytes, uint64(n))
}

func (c *ConnCommonFields) NewPendingChannel() (channel int) {
	panic("not implemented")
}

func (c *ConnCommonFields) DeletePendingChannel(channel int) {
	panic("not implemented")
}

func (c *ConnCommonFields) WriteToChannel(channel int, bytes []byte) (err error) {
	panic("not implemented")
}

func (c *ConnCommonFields) SetCrypto(crypto *Crypto) {
	c.crypto.Store(crypto)
	c.cryptoCond.Broadcast()
}

func (c *ConnCommonFields) GetCrypto() *Crypto {
	x := c.crypto.Load()
	if x == nil {
		return nil
	}
	return x.(*Crypto)
}

func (c *ConnCommonFields) MustGetCrypto() *Crypto {
	var v interface{}
	for v = c.crypto.Load(); v == nil; v = c.crypto.Load() {
		c.cryptoMutex.Lock()
		c.cryptoCond.Wait()
		c.cryptoMutex.Unlock()
	}
	return v.(*Crypto)
}
