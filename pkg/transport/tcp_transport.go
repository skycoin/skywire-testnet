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
)

// ErrUnknownRemote returned for connection attempts for remotes
// missing from the translation table.
var ErrUnknownRemote = errors.New("unknown remote")

// TCPFactory implements Factory over TCP connection.
type TCPFactory struct {
	Pk      cipher.PubKey
	PkTable PubKeyTable
	Lsr     *net.TCPListener
}

// NewTCPFactory constructs a new TCP Factory
func NewTCPFactory(pk cipher.PubKey, pubkeysFile string, tcpAddr string) (Factory, error) {

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

	return &TCPFactory{pk, pkTbl, tcpListener}, nil
}

// Accept accepts a remotely-initiated Transport.
func (f *TCPFactory) Accept(ctx context.Context) (Transport, error) {
	conn, err := f.Lsr.AcceptTCP()
	if err != nil {
		return nil, err
	}

	raddr := conn.RemoteAddr().(*net.TCPAddr)
	rpk := f.PkTable.RemotePK(raddr.String())
	if rpk.Null() {
		return nil, fmt.Errorf("error: %v, raddr: %v, rpk: %v", ErrUnknownRemote, raddr.String(), rpk)
	}

	return &TCPTransport{conn, [2]cipher.PubKey{f.Pk, rpk}}, nil
}

// Dial initiates a Transport with a remote node.
func (f *TCPFactory) Dial(ctx context.Context, remote cipher.PubKey) (Transport, error) {
	addr := f.PkTable.RemoteAddr(remote)
	if addr == "" {
		return nil, ErrUnknownRemote
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	lsnAddr, err := net.ResolveTCPAddr("tcp", f.Lsr.Addr().String())
	if err != nil {
		return nil, fmt.Errorf("error in resolving local address")
	}
	locAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", lsnAddr.IP.String(), "0"))
	if err != nil {
		return nil, fmt.Errorf("error in constructing local address ")
	}

	conn, err := net.DialTCP("tcp", locAddr, tcpAddr)
	if err != nil {
		return nil, err
	}

	return &TCPTransport{conn, [2]cipher.PubKey{f.Pk, remote}}, nil
}

// Close implements io.Closer
func (f *TCPFactory) Close() error {
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
	edges [2]cipher.PubKey
}

// Edges returns the  TCPTransport edges.
func (tr *TCPTransport) Edges() [2]cipher.PubKey {
	return SortEdges(tr.edges)
}

// Type returns the string representation of the transport type.
func (tr *TCPTransport) Type() string {
	return "tcp"
}

// PubKeyTable provides translation between remote PubKey and TCPAddr.
type PubKeyTable interface {
	RemoteAddr(remotePK cipher.PubKey) string
	RemotePK(address string) cipher.PubKey
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

// MemoryPKTable returns in memory implementation of the PubKeyTable.
func MemoryPubKeyTable(entries map[cipher.PubKey]string) PubKeyTable {
	return memoryPubKeyTable(entries)
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
