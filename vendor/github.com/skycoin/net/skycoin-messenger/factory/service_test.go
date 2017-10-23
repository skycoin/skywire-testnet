package factory

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

func newTestConnection() *Connection {
	connection := &Connection{}
	connection.keySetCond = sync.NewCond(connection.fieldsMutex.RLocker())
	return connection
}

func TestRegisterAndFind(t *testing.T) {
	conn1 := newTestConnection()
	connkey1 := cipher.PubKey([33]byte{0x01})
	key1 := cipher.PubKey([33]byte{0xf1})
	subs1 := []*Service{{Key: key1, Attributes: []string{"vpn"}},
		{Key: cipher.PubKey([33]byte{0xf2}), Attributes: []string{"vpn"}}}
	conn1.SetKey(connkey1)
	service := newServiceDiscovery()
	service.register(conn1, &NodeServices{Services: subs1})

	var result []cipher.PubKey
	result = service.find(key1)
	if len(result) != 1 || result[0] != connkey1 {
		t.Fatalf("len(result) != 1 || result[0] != connkey1 %v", result)
	}
	resultOfAttrs := service.findByAttributes("vpn")
	if len(resultOfAttrs) != 1 || result[0] != connkey1 {
		t.Fatalf("len(result) != 1 || result[0] != connkey1 %v", result)
	}

	conn2 := newTestConnection()
	connkey2 := cipher.PubKey([33]byte{0x02})
	key2 := cipher.PubKey([33]byte{0xa1})
	subs2 := []*Service{{Key: key2, Attributes: []string{"ss"}},
		{Key: key1, Attributes: []string{"ss"}}}
	conn2.SetKey(connkey2)

	service.register(conn2, &NodeServices{Services: subs2})

	result = service.find(key1)
	if len(result) != 2 {
		t.Fatalf("len(result) != 2 %v", result)
	}
	resultOfAttrs = service.findByAttributes("a")
	if len(resultOfAttrs) != 0 {
		t.Fatalf("len(result) != 0 %v", result)
	}
	resultOfAttrs = service.findByAttributes("vpn")
	if len(resultOfAttrs) != 2 {
		t.Fatalf("len(result) != 2 %v", result)
	}
	resultOfAttrs = service.findByAttributes("ss")
	if len(resultOfAttrs) != 2 {
		t.Fatalf("len(result) != 2 %v", result)
	}

	conn3 := newTestConnection()
	connkey3 := cipher.PubKey([33]byte{0x03})
	subs3 := []*Service{
		{Key: cipher.PubKey([33]byte{0xff}), Attributes: []string{"vpn"}}}
	conn3.SetKey(connkey3)

	service.register(conn3, &NodeServices{Services: subs3})

	resultOfAttrs = service.findByAttributes("vpn")
	if len(resultOfAttrs) != 3 {
		t.Fatalf("len(result) != 3 %v", result)
	}

	resultOfAttrs = service.findByAttributes("vpn", "a")
	if len(resultOfAttrs) != 0 {
		t.Fatalf("len(result) != 0 %v", result)
	}

	if len(service.subscription2Subscriber) != 4 {
		t.Fatal(service.subscription2Subscriber)
	}
	service.unregister(conn3)
	if len(service.subscription2Subscriber) != 3 {
		t.Fatal(service.subscription2Subscriber)
	}
	service.unregister(conn2)
	if len(service.subscription2Subscriber) != 2 {
		t.Fatal(service.subscription2Subscriber)
	}
	service.unregister(conn1)
	if len(service.subscription2Subscriber) != 0 {
		t.Fatal(service.subscription2Subscriber)
	}
	if len(service.attribute2Keys) != 0 {
		t.Fatal(service.attribute2Keys)
	}
	if len(service.key2Attributes) != 0 {
		t.Fatal(service.key2Attributes)
	}
}
