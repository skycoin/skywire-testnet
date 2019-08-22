package transport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	th "github.com/skycoin/skywire/internal/testhelpers"
)

// ErrUnknownRemote returned for connection attempts for remotes
// missing from the translation table.
var ErrUnknownRemote = errors.New("unknown remote")

// TCPFactory implements Factory over TCP connection.
type TCPFactory struct {
	Pk      cipher.PubKey
	PkTable PubKeyTable
	Lsr     *net.TCPListener
	Logger  *logging.Logger
}

// NewTCPFactory constructs a new TCP Factory
func NewTCPFactory(pk cipher.PubKey, pubkeysFile string, tcpAddr string, logger *logging.Logger) (Factory, error) {
	logger.Debug(th.Trace("ENTER"))
	defer logger.Debug(th.Trace("EXIT"))

	pkTbl, err := FilePubKeyTable(pubkeysFile)
	if err != nil {
		return nil, fmt.Errorf("error %v reading %v", err, pubkeysFile)
	}

	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("error %v resolving %v", err, tcpAddr)
	}

	tcpListener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error %v listening %v", err, tcpAddr)
	}

	logger.Infof("NewTCPFactory: success, records in pubkeys file: %v\n", pkTbl.Count())
	return &TCPFactory{pk, pkTbl, tcpListener, logger}, nil
}

// Accept accepts a remotely-initiated Transport.
func (f *TCPFactory) Accept(ctx context.Context) (Transport, error) {
	f.Logger.Debug(th.Trace("ENTER"))
	defer f.Logger.Debug(th.Trace("EXIT"))

	conn, err := f.Lsr.AcceptTCP()
	if err != nil {
		f.Logger.Warnf("[TCPFactory.Accept] f.Lsr.AcceptTCP err: %v\n", err)
		return nil, err
	}

	raddr := conn.RemoteAddr().(*net.TCPAddr)
	rpk := f.PkTable.RemotePK(raddr.String())
	if rpk.Null() {
		f.Logger.Infof("[TCPFactory.Accept] f.PkTable.RemotePK rpk.Null() for %v\n", raddr)
		return nil, fmt.Errorf("error: %v, raddr: %v, rpk: %v", ErrUnknownRemote, raddr.String(), rpk)
	}
	f.Logger.Info("[TCPFactory.Accept] success")

	// return &TCPTransport{conn, [2]cipher.PubKey{f.Pk, rpk}}, nil
	return &TCPTransport{conn, f.Pk, rpk}, nil
}

// Dial initiates a Transport with a remote node.
func (f *TCPFactory) Dial(ctx context.Context, remote cipher.PubKey) (Transport, error) {
	f.Logger.Debug(th.Trace("ENTER"))
	defer f.Logger.Debug(th.Trace("EXIT"))

	addr := f.PkTable.RemoteAddr(remote)
	if addr == "" {
		f.Logger.Warnf("[TCPFactory.Dial] f.PkTable.RemoteAddr(remote) is empty for %v\n", remote)
		return nil, ErrUnknownRemote
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		f.Logger.Warnf("[TCPFactory.Dial] net.ResolveTCPAddr_tcpAddr err: %v\n", err)
		return nil, err
	}

	lsnAddr, err := net.ResolveTCPAddr("tcp", f.Lsr.Addr().String())
	if err != nil {
		f.Logger.Warnf("[TCPFactory.Dial] net.ResolveTCPAddr_lsnAddr err: %v\n", err)
		return nil, fmt.Errorf("error in resolving local address")
	}
	locAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", lsnAddr.IP.String(), "0"))
	if err != nil {
		f.Logger.Warnf("[TCPFactory.Dial] net.ResolveTCPAddr_locAddr err: %v\n", err)
		// net.ResolveTCPAddr("tcp", f.Lsr.Addr().String())
		return nil, fmt.Errorf("error in constructing local address ")
	}

	conn, err := net.DialTCP("tcp", locAddr, tcpAddr)
	if err != nil {
		f.Logger.Warnf("[TCPFactory.Dial] net.DialTCP err: %v\n", err)
		return nil, err
	}

	return &TCPTransport{conn, f.Pk, remote}, nil
}

// Close implements io.Closer
func (f *TCPFactory) Close() error {
	f.Logger.Debug(th.Trace("ENTER"))
	defer f.Logger.Debug(th.Trace("EXIT"))

	if f == nil {
		return nil
	}
	return f.Lsr.Close()
}

// Local returns the local public key.
func (f *TCPFactory) Local() cipher.PubKey {
	return f.Pk
}

// Type returns the Transport type.
func (f *TCPFactory) Type() string {
	return "tcp-transport"
}

// TCPTransport implements Transport over TCP connection.
type TCPTransport struct {
	*net.TCPConn
	localKey  cipher.PubKey
	remoteKey cipher.PubKey
}

// LocalPK returns the TCPTransport local public key.
func (tr *TCPTransport) LocalPK() cipher.PubKey {

	return tr.localKey
}

// RemotePK returns the TCPTransport remote public key.
func (tr *TCPTransport) RemotePK() cipher.PubKey {
	return tr.remoteKey
}

// Type returns the string representation of the transport type.
func (tr *TCPTransport) Type() string {
	return "tcp-transport"
}

// PubKeyTable provides translation between remote PubKey and TCPAddr.
type PubKeyTable interface {
	RemoteAddr(remotePK cipher.PubKey) string
	RemotePK(address string) cipher.PubKey
	Count() int
}

type memPKTable struct {
	entries map[cipher.PubKey]string
	reverse map[string]cipher.PubKey
}

func memoryPubKeyTable(entries map[cipher.PubKey]string) *memPKTable {
	reverse := make(map[string]cipher.PubKey)
	for k, v := range entries {
		addr, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			panic("error in resolving address")
		}
		reverse[addr.IP.String()] = k
	}
	return &memPKTable{entries, reverse}
}

// MemoryPubKeyTable returns in memory implementation of the PubKeyTable.
func MemoryPubKeyTable(entries map[cipher.PubKey]string) PubKeyTable {
	return memoryPubKeyTable(entries)
}

// Count returns number of Table records
func (t *memPKTable) Count() int {
	return len(t.entries)
}

func (t *memPKTable) RemoteAddr(remotePK cipher.PubKey) string {
	return t.entries[remotePK]
}

func (t *memPKTable) RemotePK(address string) cipher.PubKey {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		panic("net.ResolveTCPAddr")
	}
	return t.reverse[addr.IP.String()]
}

type filePKTable struct {
	dbFile string
	*memPKTable
}

// FilePubKeyTable returns file based implementation of the PubKeyTable.
func FilePubKeyTable(dbFile string) (PubKeyTable, error) {

	path, err := filepath.Abs(dbFile)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	entries := make(map[cipher.PubKey]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		components := strings.Fields(scanner.Text())
		if len(components) != 2 {
			continue
		}

		pk := cipher.PubKey{}
		if err := pk.UnmarshalText([]byte(components[0])); err != nil {
			continue
		}

		addr, err := net.ResolveTCPAddr("tcp", components[1])
		if err != nil {
			continue
		}

		entries[pk] = addr.String()
	}

	return &filePKTable{dbFile, memoryPubKeyTable(entries)}, nil
}
