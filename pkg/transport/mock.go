package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/snet"
)

// ErrTransportCommunicationTimeout represent timeout error for a mock transport.
var ErrTransportCommunicationTimeout = errors.New("transport communication operation timed out")

type fConn struct {
	net.Conn
	cipher.PubKey
}

// MockFactory implements Factory over net.Pipe connections.
type MockFactory struct {
	local   cipher.PubKey
	inDone  chan struct{}
	outDone chan struct{}
	in      chan *fConn
	out     chan *fConn
	fType   string
}

// NewMockFactoryPair constructs a pair of MockFactories.
func NewMockFactoryPair(local, remote cipher.PubKey) (*MockFactory, *MockFactory) {
	var (
		inDone  = make(chan struct{})
		outDone = make(chan struct{})
		in      = make(chan *fConn)
		out     = make(chan *fConn)
	)
	a := &MockFactory{local, inDone, outDone, in, out, "mock"}
	b := &MockFactory{remote, outDone, inDone, out, in, "mock"}
	return a, b
}

// SetType sets type of transport.
func (f *MockFactory) SetType(fType string) {
	f.fType = fType
}

// Accept waits for new net.Conn notification from another MockFactory.
func (f *MockFactory) Accept(ctx context.Context) (*MockTransport, error) {
	select {
	case conn, ok := <-f.in:
		if !ok {
			return nil, errors.New("factory: closed")
		}
		return NewMockTransport(conn, f.local, conn.PubKey), nil

	case <-f.inDone:
		return nil, errors.New("factory: closed")
	}
}

// Dial creates pair of net.Conn via net.Pipe and passes one end to another MockFactory.
func (f *MockFactory) Dial(ctx context.Context, remote cipher.PubKey) (*MockTransport, error) {
	in, out := net.Pipe()
	select {
	case <-f.outDone:
		return nil, errors.New("factory: closed")
	case f.out <- &fConn{in, f.local}:
		return NewMockTransport(out, f.local, remote), nil
	}
}

// Close closes notification channel between a pair of MockFactories.
func (f *MockFactory) Close() error {
	if f == nil {
		return nil
	}
	select {
	case <-f.inDone:
	default:
		close(f.inDone)
	}
	return nil
}

// Local returns a local PubKey of the Factory.
func (f *MockFactory) Local() cipher.PubKey {
	return f.local
}

// Type returns type of the Factory.
func (f *MockFactory) Type() string {
	return f.fType
}

// MockTransport is a transport that accepts custom writers and readers to use them in Read and Write
// operations
type MockTransport struct {
	rw        io.ReadWriteCloser
	localKey  cipher.PubKey
	remoteKey cipher.PubKey
	context   context.Context
}

// NewMockTransport creates a transport with the given secret key and remote public key, taking a writer
// and a reader that will be used in the Write and Read operation
func NewMockTransport(rw io.ReadWriteCloser, local, remote cipher.PubKey) *MockTransport {
	return &MockTransport{rw, local, remote, context.Background()}
}

// Read implements reader for mock transport
func (m *MockTransport) Read(p []byte) (n int, err error) {
	select {
	case <-m.context.Done():
		return 0, ErrTransportCommunicationTimeout
	default:
		return m.rw.Read(p)
	}
}

// Write implements writer for mock transport
func (m *MockTransport) Write(p []byte) (n int, err error) {
	select {
	case <-m.context.Done():
		return 0, ErrTransportCommunicationTimeout
	default:
		return m.rw.Write(p)
	}
}

// Close implements closer for mock transport
func (m *MockTransport) Close() error {
	if m == nil {
		return nil
	}
	return m.rw.Close()
}

// LocalPK returns local public key of MockTransport
func (m *MockTransport) LocalPK() cipher.PubKey {
	return m.localKey
}

// RemotePK returns remote public key of MockTransport
func (m *MockTransport) RemotePK() cipher.PubKey {
	return m.remoteKey
}

// SetDeadline sets a deadline for the write/read operations of the mock transport
func (m *MockTransport) SetDeadline(t time.Time) error {
	ctx, cancel := context.WithDeadline(m.context, t)
	m.context = ctx

	go func(cancel context.CancelFunc) {
		time.Sleep(time.Until(t))
		cancel()
	}(cancel)

	return nil
}

// Type returns the type of the mock transport
func (m *MockTransport) Type() string {
	return "mock"
}

// MockTransportManagersPair constructs a pair of Transport Managers
func MockTransportManagersPair() (pk1, pk2 cipher.PubKey, m1, m2 *Manager, errCh chan error, err error) {
	discovery := NewDiscoveryMock()
	logs := InMemoryTransportLogStore()

	var sk1, sk2 cipher.SecKey
	pk1, sk1 = cipher.GenerateKeyPair()
	pk2, sk2 = cipher.GenerateKeyPair()

	mc1 := &ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: discovery, LogStore: logs}
	mc2 := &ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: discovery, LogStore: logs}

	//f1, f2 := NewMockFactoryPair(pk1, pk2)

	nc1 := snet.Config{PubKey: pk1, SecKey: sk1}
	nc2 := snet.Config{PubKey: pk2, SecKey: sk2}

	net1 := snet.New(nc1)
	net2 := snet.New(nc2)

	if m1, err = NewManager(net1, mc1); err != nil {
		return
	}
	if m2, err = NewManager(net2, mc2); err != nil {
		return
	}

	go m1.Serve(context.TODO())
	go m2.Serve(context.TODO())

	return
}

// MockTransportManager creates Manager
func MockTransportManager() (cipher.PubKey, *Manager, error) {
	_, pkB, mgrA, _, _, err := MockTransportManagersPair()
	return pkB, mgrA, err
}
