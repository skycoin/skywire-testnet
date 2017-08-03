package conn

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/msg"
	"math/big"
	"sync"
	"time"
)

type PendingMap struct {
	Pending map[uint32]*msg.Message
	sync.RWMutex
	ackedMessages        map[uint32]*msg.Message
	ackedMessagesMutex   sync.RWMutex
	lastMinuteAcked      map[uint32]*msg.Message
	lastMinuteAckedMutex sync.RWMutex

	statistics  string
	fieldsMutex sync.RWMutex
	logger      *log.Entry
}

func NewPendingMap(logger *log.Entry) *PendingMap {
	pendingMap := &PendingMap{Pending: make(map[uint32]*msg.Message), ackedMessages: make(map[uint32]*msg.Message)}
	pendingMap.logger = logger
	go pendingMap.analyse()
	return pendingMap
}

func (m *PendingMap) AddMsg(k uint32, v *msg.Message) {
	m.Lock()
	m.Pending[k] = v
	m.Unlock()
	v.Transmitted()
}

func (m *PendingMap) DelMsg(k uint32) {
	m.RLock()
	v, ok := m.Pending[k]
	m.RUnlock()

	if !ok {
		return
	}

	v.Acked()

	m.ackedMessagesMutex.Lock()
	m.ackedMessages[k] = v
	m.ackedMessagesMutex.Unlock()

	m.Lock()
	delete(m.Pending, k)
	m.logger.Debugf("acked %d, Pending:%d, %v", k, len(m.Pending), m.Pending)
	m.Unlock()
}

func (m *PendingMap) analyse() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			m.ackedMessagesMutex.Lock()
			m.lastMinuteAckedMutex.Lock()
			m.lastMinuteAcked = m.ackedMessages
			m.lastMinuteAckedMutex.Unlock()
			m.ackedMessages = make(map[uint32]*msg.Message)
			m.ackedMessagesMutex.Unlock()

			m.lastMinuteAckedMutex.RLock()
			if len(m.lastMinuteAcked) < 1 {
				m.lastMinuteAckedMutex.RUnlock()
				continue
			}
			var max, min int64
			sum := new(big.Int)
			bytesSent := 0
			for _, v := range m.lastMinuteAcked {
				latency := v.Latency.Nanoseconds()
				if max < latency {
					max = latency
				}
				if min == 0 || min > latency {
					min = latency
				}
				y := new(big.Int)
				y.SetInt64(latency)
				sum.Add(sum, y)

				bytesSent += v.TotalSize()
			}
			n := new(big.Int)
			n.SetInt64(int64(len(m.lastMinuteAcked)))
			avg := new(big.Int)
			avg.Div(sum, n)
			m.lastMinuteAckedMutex.RUnlock()

			m.fieldsMutex.Lock()
			m.statistics = fmt.Sprintf("sent: %d bytes, latency: max %d ns, min %d ns, avg %s ns, count %s", bytesSent, max, min, avg, n)
			m.fieldsMutex.Unlock()
			m.logger.Debug(m.statistics)
		}
	}
}
