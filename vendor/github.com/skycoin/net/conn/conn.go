package conn

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
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
}

type ConnCommonFields struct {
	*PendingMap
	seq uint32 // id of last message, increment every new message

	HighestACKedSequenceNumber uint32 // highest packet that has been ACKed
	LastAck                    int64  // last time an ACK of receipt was received (better to store id of highest packet id with an ACK?)

	Status int // STATUS_CONNECTING, STATUS_CONNECTED, STATUS_ERROR
	Err    error

	In          chan []byte
	Out         chan []byte
	closed      bool
	fieldsMutex sync.RWMutex
	writeMutex  sync.Mutex

	CTXLogger *log.Entry
}

func NewConnCommonFileds() ConnCommonFields {
	entry := log.WithField("ctxId", atomic.AddUint32(&ctxId, 1))
	return ConnCommonFields{
		CTXLogger:  entry,
		PendingMap: NewPendingMap(entry), In: make(chan []byte), Out: make(chan []byte)}
}

func (c *ConnCommonFields) SetStatusToConnected() {
	c.fieldsMutex.Lock()
	c.Status = STATUS_CONNECTED
	c.fieldsMutex.Unlock()
}

func (c *ConnCommonFields) SetStatusToError(err error) {
	c.fieldsMutex.Lock()
	c.Status = STATUS_ERROR
	c.Err = err
	c.fieldsMutex.Unlock()
	c.CTXLogger.Debugf("SetStatusToError %v", err)
}

func (c *ConnCommonFields) UpdateLastAck(s uint32) {
	c.fieldsMutex.Lock()
	c.LastAck = time.Now().Unix()
	if s > c.HighestACKedSequenceNumber {
		c.HighestACKedSequenceNumber = s
	}
	c.fieldsMutex.Unlock()
}

func (c *ConnCommonFields) GetContextLogger() *log.Entry {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	return c.CTXLogger
}

func (c *ConnCommonFields) SetContextLogger(l *log.Entry) {
	c.fieldsMutex.Lock()
	c.CTXLogger = l
	c.fieldsMutex.Unlock()
}

func (c *ConnCommonFields) Close() {
	defer func() {
		if err := recover(); err != nil {
			c.CTXLogger.Debug("closing closed udpconn")
		}
	}()
	c.fieldsMutex.Lock()
	if c.closed {
		c.fieldsMutex.Unlock()
		return
	}
	c.closed = true
	c.fieldsMutex.Unlock()

	close(c.In)
	close(c.Out)
}

func (c *ConnCommonFields) IsClosed() bool {
	c.fieldsMutex.RLock()
	defer c.fieldsMutex.RUnlock()
	return c.closed
}
