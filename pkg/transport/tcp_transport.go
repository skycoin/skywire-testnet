package transport

import (
	"bufio"
	"context"
	"errors"
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
	l   *net.TCPListener
	lpk cipher.PubKey
	pkt PubKeyTable
}

// NewTCPFactory constructs a new TCP Factory.
func NewTCPFactory(lpk cipher.PubKey, pkt PubKeyTable, l *net.TCPListener) Factory {
	return &TCPFactory{l, lpk, pkt}
}

// Accept accepts a remotely-initiated Transport.
func (f *TCPFactory) Accept(ctx context.Context) (Transport, error) {
	conn, err := f.l.AcceptTCP()
	if err != nil {
		return nil, err
	}

	raddr := conn.RemoteAddr().(*net.TCPAddr)
	rpk := f.pkt.RemotePK(raddr.IP)
	if rpk.Null() {
		return nil, ErrUnknownRemote
	}

	return &TCPTransport{conn, f.lpk, rpk}, nil
}

// Dial initiates a Transport with a remote node.
func (f *TCPFactory) Dial(ctx context.Context, remote cipher.PubKey) (Transport, error) {
	raddr := f.pkt.RemoteAddr(remote)
	if raddr == nil {
		return nil, ErrUnknownRemote
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		return nil, err
	}

	return &TCPTransport{conn, f.lpk, remote}, nil
}

// Close implements io.Closer
func (f *TCPFactory) Close() error {
	if f == nil {
		return nil
	}
	return f.l.Close()
}

// Local returns the local public key.
func (f *TCPFactory) Local() cipher.PubKey {
	return f.lpk
}

// Type returns the Transport type.
func (f *TCPFactory) Type() string {
	return "tcp"
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
	return "tcp"
}

// PubKeyTable provides translation between remote PubKey and TCPAddr.
type PubKeyTable interface {
	RemoteAddr(remotePK cipher.PubKey) *net.TCPAddr
	RemotePK(remoteIP net.IP) cipher.PubKey
}

type inMemoryPKTable struct {
	entries map[cipher.PubKey]*net.TCPAddr
}

// InMemoryPubKeyTable returns in memory implementation of the PubKeyTable.
func InMemoryPubKeyTable(entries map[cipher.PubKey]*net.TCPAddr) PubKeyTable {
	return &inMemoryPKTable{entries}
}

func (t *inMemoryPKTable) RemoteAddr(remotePK cipher.PubKey) *net.TCPAddr {
	return t.entries[remotePK]
}

func (t *inMemoryPKTable) RemotePK(remoteIP net.IP) cipher.PubKey {
	for pk, addr := range t.entries {
		if addr.IP.String() == remoteIP.String() {
			return pk
		}
	}

	return cipher.PubKey{}
}

type filePKTable struct {
	dbFile *os.File
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

	return &filePKTable{f}, nil
}

func (t *filePKTable) RemoteAddr(remotePK cipher.PubKey) *net.TCPAddr {
	var raddr *net.TCPAddr
	t.Seek(func(pk cipher.PubKey, addr *net.TCPAddr) bool {
		if pk == remotePK {
			raddr = addr
			return true
		}

		return false
	})
	return raddr
}

func (t *filePKTable) RemotePK(remoteIP net.IP) cipher.PubKey {
	var rpk cipher.PubKey
	t.Seek(func(pk cipher.PubKey, addr *net.TCPAddr) bool {
		if remoteIP.String() == addr.IP.String() {
			rpk = pk
			return true
		}

		return false
	})
	return rpk
}

func (t *filePKTable) Seek(seekFunc func(pk cipher.PubKey, addr *net.TCPAddr) bool) {
	defer func() {
		if _, err := t.dbFile.Seek(0, 0); err != nil {
			log.WithError(err).Warn("Failed to seek to the beginning of DB")
		}
	}()

	scanner := bufio.NewScanner(t.dbFile)
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

		if seekFunc(pk, addr) {
			return
		}
	}
}
