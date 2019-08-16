package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/skywire/pkg/routing"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
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

	lSK cipher.SecKey
	rPK cipher.PubKey

	fac Factory
	dc  DiscoveryClient
	ls  LogStore

	Entry      Entry
	LogEntry   *LogEntry
	logUpdates uint32

	conn   Transport
	connCh chan struct{}
	connMx sync.Mutex

	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// NewManagedTransport creates a new ManagedTransport.
func NewManagedTransport(fac Factory, dc DiscoveryClient, ls LogStore, rPK cipher.PubKey, lSK cipher.SecKey) *ManagedTransport {
	mt := &ManagedTransport{
		log:      logging.MustGetLogger(fmt.Sprintf("tp:%s", rPK.String()[:6])),
		lSK:      lSK,
		rPK:      rPK,
		fac:      fac,
		dc:       dc,
		ls:       ls,
		Entry:    makeEntry(fac.Local(), rPK, dmsg.Type),
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
			_ = mt.conn.Close() //nolint:errcheck
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
func (mt *ManagedTransport) Accept(ctx context.Context, tp Transport) error {
	mt.connMx.Lock()
	defer mt.connMx.Unlock()

	if !mt.isServing() {
		_ = tp.Close() //nolint:errcheck
		return ErrNotServing
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := MakeSettlementHS(false).Do(ctx, mt.dc, tp, mt.lSK); err != nil {
		return fmt.Errorf("settlement handshake failed: %v", err)
	}

	return mt.setIfConnNil(ctx, tp)
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
	tp, err := mt.fac.Dial(ctx, mt.rPK)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := MakeSettlementHS(true).Do(ctx, mt.dc, tp, mt.lSK); err != nil {
		return fmt.Errorf("settlement handshake failed: %v", err)
	}

	return mt.setIfConnNil(ctx, tp)
}

func (mt *ManagedTransport) getConn() Transport {
	mt.connMx.Lock()
	conn := mt.conn
	mt.connMx.Unlock()
	return conn
}

// sets conn if `mt.conn` is nil otherwise, closes the conn.
// TODO: Add logging here.
func (mt *ManagedTransport) setIfConnNil(ctx context.Context, conn Transport) error {
	if mt.conn != nil {
		_ = conn.Close() //nolint:errcheck
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
		_ = mt.conn.Close() //nolint:errcheck
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
	var conn Transport
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
func (mt *ManagedTransport) Type() string { return mt.fac.Type() }
