package router

/*

import (
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/routing"
)


type portManager struct {
	ports  *portList
	logger *logging.Logger
}

type portList struct {
	sync.Mutex
	minPort routing.Port //type Port uint16
	ports   map[routing.Port]*portBind
}

type portBind struct {
	conn  *appProtocol
	loops *loopList
}

type appProtocol struct {
	rw    io.ReadWriteCloser
	chans *appChanList
}

type appChanList struct {
	sync.Mutex
	chans map[byte]chan []byte
}

type loopList struct {
	sync.Mutex
	loops map[routing.Addr]loop // key: remote address (pk+port), value: forwarding transport and route ID.
}

// Addr is routing.Addr
type Addr struct {
	PubKey cipher.PubKey `json:"pk"`
	Port   routing.Port  `json:"port"` //uint16
}


type loop struct {
	trID    uuid.UUID       //forwarding transport
	routeID routing.RouteID // type RouteID uint32
}
*/