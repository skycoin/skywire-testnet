package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/snet"
)

const logWriteInterval = time.Second * 3

// Records number of managedTransports.
var mTpCount int32

var (
	// ErrNotServing is the error returned when a transport is no longer served.
	ErrNotServing = errors.New("transport is no longer being served")

	// ErrConnAlreadyExists occurs when an underlying transport connection already exists.
	ErrConnAlreadyExists = errors.New("underlying transport connection already exists")
)

// ManagedTransport manages a direct line of communication between two visor nodes.
// It is made up of two underlying uni-directional connections.
type ManagedTransport struct {
	log *logging.Logger

	rPK        cipher.PubKey
	netName    string
	Entry      Entry
	LogEntry   *LogEntry
	logUpdates uint32

	dc DiscoveryClient
	ls LogStore

	n      *snet.Network
	conn   *snet.Conn
	connCh chan struct{}
	connMx sync.Mutex

	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewManagedTransport creates a new ManagedTransport.
func NewManagedTransport(n *snet.Network, dc DiscoveryClient, ls LogStore, rPK cipher.PubKey, netName string) *ManagedTransport {
	mt := &ManagedTransport{
		log:      logging.MustGetLogger(fmt.Sprintf("tp:%s", rPK.String()[:6])),
		rPK:      rPK,
		netName:  netName,
		n:        n,
		dc:       dc,
		ls:       ls,
		Entry:    makeEntry(n.LocalPK(), rPK, netName),
		LogEntry: new(LogEntry),
		connCh:   make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	mt.wg.Add(2)
	return mt
}

// Serve serves and manages the transport.
func (mt *ManagedTransport) Serve(readCh chan<- routing.Packet, done <-chan struct{}) {
	defer mt.wg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-mt.done
		cancel()
	}()

	logTicker := time.NewTicker(logWriteInterval)
	defer logTicker.Stop()

	mt.log.Infof("serving: tpID(%v) rPK(%s) srvQty[%d]", mt.Entry.ID, mt.rPK, atomic.AddInt32(&mTpCount, 1))
	defer mt.log.Infof("stopped: tpID(%v) rPK(%s) srvQty[%d]", mt.Entry.ID, mt.rPK, atomic.AddInt32(&mTpCount, -1))

	defer func() {
		// Ensure logs tp logs are up to date before closing.
		if mt.logMod() {
			if err := mt.ls.Record(mt.Entry.ID, mt.LogEntry); err != nil {
				mt.log.Warnf("Failed to record log entry: %s", err)
			}
		}

		// End connection.
		mt.connMx.Lock()
		close(mt.connCh)
		if mt.conn != nil {
			if err := mt.conn.Close(); err != nil {
				mt.log.WithError(err).Warn("Failed to close connection")
			}
			mt.conn = nil
		}
		mt.connMx.Unlock()
	}()

	go func() {
		defer func() {
			mt.log.Infof("closed readPacket loop.")
			mt.wg.Done()
		}()
		for {
			p, err := mt.readPacket()
			if err != nil {
				if err == ErrNotServing {
					return
				}
				mt.connMx.Lock()
				mt.clearConn(ctx)
				mt.connMx.Unlock()
				mt.log.Warnf("failed to read packet: %v", err)
				continue
			}
			select {
			case <-done:
				return
			case readCh <- p:
			}
		}
	}()

	for {
		select {
		case <-mt.done:
			return

		case <-logTicker.C:
			if mt.logMod() {
				if err := mt.ls.Record(mt.Entry.ID, mt.LogEntry); err != nil {
					mt.log.Warnf("Failed to record log entry: %s", err)
				}
			} else {
				// If there has not been any activity, ensure underlying 'write' tp is still up.
				mt.connMx.Lock()
				if mt.conn == nil {
					if err := mt.dial(ctx); err != nil {
						mt.log.Warnf("failed to redial underlying connection: %v", err)
					}
				}
				mt.connMx.Unlock()
			}
		}
	}
}

func (mt *ManagedTransport) isServing() bool {
	select {
	case <-mt.done:
		return false
	default:
		return true
	}
}

// Close stops serving the transport.
func (mt *ManagedTransport) Close() {
	if mt.close() {
		// Update transport entry.
		if _, err := mt.dc.UpdateStatuses(context.Background(), &Status{ID: mt.Entry.ID, IsUp: false}); err != nil {
			mt.log.Warnf("Failed to update transport status: %s", err)
		}
	}
}

func (mt *ManagedTransport) close() (closed bool) {
	mt.once.Do(func() {
		close(mt.done)
		mt.wg.Wait()
		closed = true
	})
	return closed
}

// Accept accepts a new underlying connection.
func (mt *ManagedTransport) Accept(ctx context.Context, conn *snet.Conn) error {
	mt.connMx.Lock()
	defer mt.connMx.Unlock()

	if conn.Network() != mt.netName {
		return errors.New("wrong network") // TODO: Make global var.
	}

	if !mt.isServing() {
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
		return ErrNotServing
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := MakeSettlementHS(false).Do(ctx, mt.dc, conn, mt.n.LocalSK()); err != nil {
		return fmt.Errorf("settlement handshake failed: %v", err)
	}

	return mt.setIfConnNil(ctx, conn)
}

// Dial dials a new underlying connection.
func (mt *ManagedTransport) Dial(ctx context.Context) error {
	mt.connMx.Lock()
	defer mt.connMx.Unlock()

	if !mt.isServing() {
		return ErrNotServing
	}

	if mt.conn != nil {
		return nil
	}
	return mt.dial(ctx)
}

// TODO: Figure out where this fella is called.
func (mt *ManagedTransport) dial(ctx context.Context) error {
	tp, err := mt.n.Dial(mt.netName, mt.rPK, snet.TransportPort)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := MakeSettlementHS(true).Do(ctx, mt.dc, tp, mt.n.LocalSK()); err != nil {
		return fmt.Errorf("settlement handshake failed: %v", err)
	}

	return mt.setIfConnNil(ctx, tp)
}

func (mt *ManagedTransport) getConn() *snet.Conn {
	mt.connMx.Lock()
	conn := mt.conn
	mt.connMx.Unlock()
	return conn
}

// sets conn if `mt.conn` is nil otherwise, closes the conn.
// TODO: Add logging here.
func (mt *ManagedTransport) setIfConnNil(ctx context.Context, conn *snet.Conn) error {
	if mt.conn != nil {
		if err := conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
		return ErrConnAlreadyExists
	}

	var err error
	for i := 0; i < 3; i++ {
		if _, err = mt.dc.UpdateStatuses(ctx, &Status{ID: mt.Entry.ID, IsUp: true}); err != nil {
			mt.log.Warnf("Failed to update transport status: %s, retrying...", err)
			continue
		}
		mt.log.Infoln("Status updated: UP")
		break
	}

	mt.conn = conn
	select {
	case mt.connCh <- struct{}{}:
	default:
	}
	return nil
}

func (mt *ManagedTransport) clearConn(ctx context.Context) {
	if mt.conn != nil {
		if err := mt.conn.Close(); err != nil {
			log.WithError(err).Warn("Failed to close connection")
		}
		mt.conn = nil
	}
	if _, err := mt.dc.UpdateStatuses(ctx, &Status{ID: mt.Entry.ID, IsUp: false}); err != nil {
		mt.log.Warnf("Failed to update transport status: %s", err)
	}
	mt.log.Infoln("Status updated: DOWN")
}

// WritePacket writes a packet to the remote.
func (mt *ManagedTransport) WritePacket(ctx context.Context, rtID routing.RouteID, payload []byte) error {
	mt.connMx.Lock()
	defer mt.connMx.Unlock()

	if !mt.isServing() {
		return ErrNotServing
	}

	if mt.conn == nil {
		if err := mt.dial(ctx); err != nil {
			return fmt.Errorf("failed to redial underlying connection: %v", err)
		}
	}

	n, err := mt.conn.Write(routing.MakePacket(rtID, payload))
	if err != nil {
		mt.clearConn(ctx)
		return err
	}
	if n > routing.PacketHeaderSize {
		mt.logSent(uint64(n - routing.PacketHeaderSize))
	}
	return nil
}

// WARNING: Not thread safe.
func (mt *ManagedTransport) readPacket() (packet routing.Packet, err error) {
	var conn *snet.Conn
	for {
		if conn = mt.getConn(); conn != nil {
			break
		}
		select {
		case <-mt.done:
			return nil, ErrNotServing
		case <-mt.connCh:
		}
	}

	h := make(routing.Packet, routing.PacketHeaderSize)
	if _, err = io.ReadFull(conn, h); err != nil {
		return nil, err
	}
	p := make([]byte, h.Size())
	if _, err = io.ReadFull(conn, p); err != nil {
		return nil, err
	}
	packet = append(h, p...)
	if n := len(packet); n > routing.PacketHeaderSize {
		mt.logRecv(uint64(n - routing.PacketHeaderSize))
	}
	mt.log.Infof("recv packet: rtID(%d) size(%d)", packet.RouteID(), packet.Size())
	return packet, nil
}

/*
	TRANSPORT LOGGING
*/

func (mt *ManagedTransport) logSent(b uint64) {
	mt.LogEntry.AddSent(b)
	atomic.AddUint32(&mt.logUpdates, 1)
}

func (mt *ManagedTransport) logRecv(b uint64) {
	mt.LogEntry.AddRecv(b)
	atomic.AddUint32(&mt.logUpdates, 1)
}

func (mt *ManagedTransport) logMod() bool {
	if ops := atomic.SwapUint32(&mt.logUpdates, 0); ops > 0 {
		mt.log.Infof("entry log: recording %d operations", ops)
		return true
	}
	return false
}

// Remote returns the remote public key.
func (mt *ManagedTransport) Remote() cipher.PubKey { return mt.rPK }

// Type returns the transport type.
func (mt *ManagedTransport) Type() string { return mt.netName }
