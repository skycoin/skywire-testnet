package websocket

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type manager struct {
	clients      map[*Client]struct{}
	clientsMutex sync.RWMutex
}

var (
	once           = &sync.Once{}
	defaultFactory *manager
	wsId           uint32
)

func getManager() *manager {
	once.Do(func() {
		defaultFactory = &manager{clients: make(map[*Client]struct{})}
		go defaultFactory.logStatus()
	})
	return defaultFactory
}

func (m *manager) newClient(c *websocket.Conn) *Client {
	logger := log.WithField("wsId", atomic.AddUint32(&wsId, 1))
	client := &Client{
		conn:       c,
		PendingMap: PendingMap{Pending: make(map[uint32]interface{})},
		push:       make(chan interface{}),
		Logger:     logger,
	}
	m.clientsMutex.Lock()
	m.clients[client] = struct{}{}
	m.clientsMutex.Unlock()
	go func() {
		client.writeLoop()
		m.clientsMutex.Lock()
		delete(m.clients, client)
		m.clientsMutex.Unlock()
	}()
	return client
}

func (m *manager) logStatus() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			m.clientsMutex.RLock()
			log.Debugf("websocket connection clients count:%d", len(m.clients))
			m.clientsMutex.RUnlock()
		}
	}
}
