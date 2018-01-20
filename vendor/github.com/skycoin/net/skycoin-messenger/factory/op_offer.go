package factory

import (
	"encoding/json"
	"net"
	"sync"
	"fmt"
)

func init() {
	ops[OP_OFFER_SERVICE] = &sync.Pool{
		New: func() interface{} {
			return new(offer)
		},
	}
}

type offer struct {
	Services *NodeServices
}

func (offer *offer) UnmarshalJSON(data []byte) (err error) {
	ss := &NodeServices{}
	err = json.Unmarshal(data, ss)
	if err != nil {
		return
	}
	if !checkNodeServices(ss) {
		err = fmt.Errorf("invalid NodeServices %#v", ss)
		return
	}
	offer.Services = ss
	return
}

func (offer *offer) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if len(offer.Services.ServiceAddress) > 0 {
		var host, port string
		_, port, err = net.SplitHostPort(offer.Services.ServiceAddress)
		if err != nil {
			return
		}
		remote := conn.GetRemoteAddr().String()
		host, _, err = net.SplitHostPort(remote)
		if err != nil {
			return
		}
		offer.Services.ServiceAddress = net.JoinHostPort(host, port)
	}
	err = f.discoveryRegister(conn, offer.Services)
	return
}
