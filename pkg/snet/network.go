package snet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
)

// Default ports.
// TODO(evanlinjin): Define these properly. These are currently random.
const (
	SetupPort      = uint16(36)  // Listening port of a setup node.
	AwaitSetupPort = uint16(136) // Listening port of a visor node for setup operations.
	TransportPort  = uint16(45)  // Listening port of a visor node for incoming transports.
)

// Network types.
const (
	DmsgType = "dmsg"
	TCPType  = "tcp"
)

var (
	// ErrUnknownNetwork occurs on attempt to dial an unknown network type.
	ErrUnknownNetwork = errors.New("unknown network type")
)

// Config represents a network configuration.
type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	TpNetworks []string // networks to be used with transports

	DmsgDiscAddr string
	DmsgMinSrvs  int

	TCPLocalAddress string
	TCPPubKeyFile   string
}

// Network represents a network between nodes in Skywire.
type Network struct {
	conf  Config
	dmsgC *dmsg.Client
	tcpF  *TCPFactory
}

// New creates a network from a config.
func New(conf Config) *Network {
	dmsgC := dmsg.NewClient(conf.PubKey,
		conf.SecKey, disc.NewHTTP(conf.DmsgDiscAddr),
		dmsg.SetLogger(logging.MustGetLogger("snet.dmsgC")))
	return &Network{
		conf:  conf,
		dmsgC: dmsgC,
	}
}

// NewRaw creates a network from a config and a dmsg client.
func NewRaw(conf Config, dmsgC *dmsg.Client) *Network {
	return &Network{
		conf:  conf,
		dmsgC: dmsgC,
	}
}

// Init initiates server connections.
func (n *Network) Init(ctx context.Context) error {
	fmt.Println("dmsg: min_servers:", n.conf.DmsgMinSrvs)
	if err := n.dmsgC.InitiateServerConnections(ctx, n.conf.DmsgMinSrvs); err != nil {
		return fmt.Errorf("failed to initiate 'dmsg': %v", err)
	}

	return nil
}

// Close closes underlying connections.
func (n *Network) Close() error {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	var dmsgErr error
	go func() {
		dmsgErr = n.dmsgC.Close()
		wg.Done()
	}()

	wg.Wait()
	if dmsgErr != nil {
		return dmsgErr
	}
	return nil
}

// LocalPK returns local public key.
func (n *Network) LocalPK() cipher.PubKey { return n.conf.PubKey }

// LocalSK returns local secure key.
func (n *Network) LocalSK() cipher.SecKey { return n.conf.SecKey }

// TransportNetworks returns network types that are used for transports.
func (n *Network) TransportNetworks() []string { return n.conf.TpNetworks }

// Dmsg returns underlying dmsg client.
func (n *Network) Dmsg() *dmsg.Client { return n.dmsgC }

// TCP returns the underlying TCPFactory.
func (n *Network) TCP() *TCPFactory { return n.tcpF }

// Dial dials a node by its public key and returns a connection.
func (n *Network) Dial(network string, pk cipher.PubKey, port uint16) (*Conn, error) {
	ctx := context.Background()
	switch network {
	case DmsgType:
		conn, err := n.dmsgC.Dial(ctx, pk, port)
		if err != nil {
			return nil, err
		}
		return makeConn(conn, network), nil
	case TCPType:
		conn, err := n.tcpF.Dial(ctx, pk)
		if err != nil {
			return nil, err
		}
		return makeConn(conn, network), nil
	default:
		return nil, ErrUnknownNetwork
	}
}

// Listen listens on the specified port.
func (n *Network) Listen(network string, port uint16) (*Listener, error) {
	switch network {
	case DmsgType:
		lis, err := n.dmsgC.Listen(port)
		if err != nil {
			return nil, err
		}
		return makeListener(lis, network), nil
	case TCPType:
		if n.conf.TCPPubKeyFile != "" {
			errMsg := func(err error) error {
				return fmt.Errorf("failed to inititiate tcp-transport: %v", err)
			}

			pkt, err := FilePubKeyTable(n.conf.TCPPubKeyFile)
			if err != nil {
				return nil, errMsg(err)
			}
			locAddr, err := net.ResolveTCPAddr("tcp", n.conf.TCPLocalAddress)
			if err != nil {
				return nil, errMsg(err)
			}
			lsn, err := net.ListenTCP("tcp", locAddr)
			if err != nil {
				return nil, errMsg(err)
			}
			n.tcpF = NewTCPFactory(n.conf.PubKey, pkt, lsn)
			return &Listener{
				Listener: lsn,
				lPK:      n.conf.PubKey,
				lPort:    666, //TODO: make something reasonable
				network:  TCPType,
			}, nil
		}
	default:
		return nil, ErrUnknownNetwork
	}
	return nil, nil
}

// Listener represents a listener.
type Listener struct {
	net.Listener
	lPK     cipher.PubKey
	lPort   uint16
	network string
}

func makeListener(l net.Listener, network string) *Listener {
	lPK, lPort := disassembleAddr(l.Addr())
	return &Listener{Listener: l, lPK: lPK, lPort: lPort, network: network}
}

// LocalPK returns a local public key of listener.
func (l Listener) LocalPK() cipher.PubKey { return l.lPK }

// LocalPort returns a local port of listener.
func (l Listener) LocalPort() uint16 { return l.lPort }

// Network returns a network of listener.
func (l Listener) Network() string { return l.network }

// AcceptConn accepts a connection from listener.
func (l Listener) AcceptConn() (*Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return makeConn(conn, l.network), nil
}

// Conn represent a connection between nodes in Skywire.
type Conn struct {
	net.Conn
	lPK     cipher.PubKey
	rPK     cipher.PubKey
	lPort   uint16
	rPort   uint16
	network string
}

func makeConn(conn net.Conn, network string) *Conn {
	lPK, lPort := disassembleAddr(conn.LocalAddr())
	rPK, rPort := disassembleAddr(conn.RemoteAddr())
	return &Conn{Conn: conn, lPK: lPK, rPK: rPK, lPort: lPort, rPort: rPort, network: network}
}

// LocalPK returns local public key of connection.
func (c Conn) LocalPK() cipher.PubKey { return c.lPK }

// RemotePK returns remote public key of connection.
func (c Conn) RemotePK() cipher.PubKey { return c.rPK }

// LocalPort returns local port of connection.
func (c Conn) LocalPort() uint16 { return c.lPort }

// RemotePort returns remote port of connection.
func (c Conn) RemotePort() uint16 { return c.rPort }

// Network returns network of connection.
func (c Conn) Network() string { return c.network }

func disassembleAddr(addr net.Addr) (pk cipher.PubKey, port uint16) {
	strs := strings.Split(addr.String(), ":")
	if len(strs) != 2 {
		panic(fmt.Errorf("network.disassembleAddr: %v %s", "invalid addr", addr.String()))
	}
	if err := pk.Set(strs[0]); err != nil {
		panic(fmt.Errorf("network.disassembleAddr: %v %s", err, addr.String()))
	}
	if strs[1] != "~" {
		if _, err := fmt.Sscanf(strs[1], "%d", &port); err != nil {
			panic(fmt.Errorf("network.disassembleAddr: %v", err))
		}
	}
	return
}
