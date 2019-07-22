package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
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
func (f *MockFactory) Accept(ctx context.Context) (Transport, error) {
	select {
	case conn, ok := <-f.in:
		if ok {
			return NewMockTransport(conn, f.local, conn.PubKey, dmsg.PurposeTest), nil
		}
	case <-f.inDone:
	}
	return nil, errors.New("factory: closed")
}

// Dial creates pair of net.Conn via net.Pipe and passes one end to another MockFactory.
func (f *MockFactory) Dial(ctx context.Context, remote cipher.PubKey, purpose string) (Transport, error) {
	in, out := net.Pipe()
	select {
	case <-f.outDone:
		return nil, errors.New("factory: closed")
	case f.out <- &fConn{in, f.local}:
		return NewMockTransport(out, f.local, remote, purpose), nil
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
	rw      io.ReadWriteCloser
	edges   [2]cipher.PubKey
	purpose string
	context context.Context
}

// NewMockTransport creates a transport with the given secret key and remote public key, taking a writer
// and a reader that will be used in the Write and Read operation
func NewMockTransport(rw io.ReadWriteCloser, local, remote cipher.PubKey, purpose string) *MockTransport {
	return &MockTransport{rw, SortPubKeys(local, remote), purpose, context.Background()}
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

// Edges returns edges of MockTransport
func (m *MockTransport) Edges() [2]cipher.PubKey {
	return SortEdges(m.edges)
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

// Purpose returns the purpose of the mock transport
func (m *MockTransport) Purpose() string {
	return dmsg.PurposeTest
}

// MockTransportManagersPair constructs a pair of Transport Managers
func MockTransportManagersPair() (pk1, pk2 cipher.PubKey, m1, m2 *Manager, errCh chan error, err error) {
	discovery := NewDiscoveryMock()
	logs := InMemoryTransportLogStore()

	var sk1, sk2 cipher.SecKey
	pk1, sk1 = cipher.GenerateKeyPair()
	pk2, sk2 = cipher.GenerateKeyPair()

	c1 := &ManagerConfig{PubKey: pk1, SecKey: sk1, DiscoveryClient: discovery, LogStore: logs}
	c2 := &ManagerConfig{PubKey: pk2, SecKey: sk2, DiscoveryClient: discovery, LogStore: logs}

	f1, f2 := NewMockFactoryPair(pk1, pk2)

	if m1, err = NewManager(c1, f1); err != nil {
		return
	}
	if m2, err = NewManager(c2, f2); err != nil {
		return
	}

	errCh = make(chan error)
	go func() { errCh <- m1.Serve(context.TODO()) }()
	go func() { errCh <- m2.Serve(context.TODO()) }()

	return
}

// MockTransportManager creates Manager
func MockTransportManager() (cipher.PubKey, *Manager, error) {
	_, pkB, mgrA, _, _, err := MockTransportManagersPair()
	return pkB, mgrA, err
}
