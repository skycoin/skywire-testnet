package factory

import (
	"container/list"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
)

type ConnectionList struct {
	list.List
	sync.RWMutex
}

func NewConnectionList() *ConnectionList {
	l := &ConnectionList{}
	l.Init()
	return l
}

func (l *ConnectionList) contain(conn *Connection) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		c, ok := e.Value.(*Connection)
		if !ok {
			continue
		}
		if c == conn {
			return e
		}
	}
	return nil
}

func (l *ConnectionList) broadcastUpdate() {
	var result []cipher.PubKey
	for e := l.Front(); e != nil; e = e.Next() {
		c, ok := e.Value.(*Connection)
		if !ok {
			continue
		}
		result = append(result, c.GetKey())
	}
	logrus.Debugf("broadcastUpdate %v", result)
	for e := l.Front(); e != nil; e = e.Next() {
		c, ok := e.Value.(*Connection)
		if !ok {
			continue
		}
		err := c.Write(GenGetServiceNodesRespMsg(result))
		if err != nil {
			c.GetContextLogger().Errorf("write GenOfferServiceRespMsg failed: %v", err)
		}
	}
}

func (l *ConnectionList) Remove(conn *Connection) int {
	l.Lock()
	defer l.Unlock()
	e := l.contain(conn)
	if e != nil {
		l.List.Remove(e)
	}
	return l.List.Len()
}

func (l *ConnectionList) PushBack(conn *Connection) {
	l.Lock()
	defer l.Unlock()
	if l.contain(conn) != nil {
		return
	}
	l.List.PushBack(conn)
}

func (l *ConnectionList) Len() int {
	l.RLock()
	defer l.RUnlock()
	return l.List.Len()
}

func (l *ConnectionList) GetPublicKeys() (result []cipher.PubKey) {
	l.RLock()
	defer l.RUnlock()
	for e := l.Front(); e != nil; e = e.Next() {
		c, ok := e.Value.(*Connection)
		if !ok {
			continue
		}
		result = append(result, c.GetKey())
	}
	return result
}
