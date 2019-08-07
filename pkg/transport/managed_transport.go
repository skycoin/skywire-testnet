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

// ErrNotServing is the error returned when a transport is no longer served.
var ErrNotServing = errors.New("transport is no longer being served")

// ManagedTransport is a wrapper transport. It stores status and ID of
// the Transport and can notify about network errors.
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

	readTp   Transport
	writeTp  Transport
	acceptCh chan Transport
	acceptMx sync.RWMutex
	dialMx   sync.Mutex

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
		acceptCh: make(chan Transport, 1),
		done:     make(chan struct{}),
	}
	mt.wg.Add(2)
	return mt
}

// Serve serves and manages the transport.
func (mt *ManagedTransport) Serve(readCh chan<- routing.Packet) {
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

		// End reading connection.
		mt.acceptMx.Lock()
		close(mt.acceptCh)
		mt.acceptCh = nil
		if mt.readTp != nil {
			_ = mt.readTp.Close() //nolint:errcheck
			mt.readTp = nil
		}
		mt.acceptMx.Unlock()

		// End writing connection.
		mt.dialMx.Lock()
		if mt.writeTp != nil {
			_ = mt.writeTp.Close() //nolint:errcheck
			mt.writeTp = nil
		}
		mt.dialMx.Unlock()
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
				mt.log.Warnf("failed to read packet: %v", err)
				continue
			}
			if !mt.isServing() {
				return
			}
			readCh <- p // TODO: data race
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
				mt.dialMx.Lock()
				if mt.writeTp == nil {
					if !mt.isServing() {
						return
					}
					if err := mt.dial(ctx); err != nil {
						mt.log.Warnf("failed to dial underlying 'write' transport: %v", err)
					}
				}
				mt.dialMx.Unlock()
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

// Accept accepts a new underlying 'read' transport (and close/replace the old one).
func (mt *ManagedTransport) Accept(ctx context.Context, tp Transport) error {
	mt.acceptMx.RLock()
	defer mt.acceptMx.RUnlock()

	if !mt.isServing() {
		_ = tp.Close() //nolint:errcheck
		return ErrNotServing
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := MakeSettlementHS(false).Do(ctx, mt.dc, tp, mt.lSK); err != nil {
		return fmt.Errorf("settlement handshake failed: %v", err)
	}

	for {
		select {
		case oldTp, ok := <-mt.acceptCh:
			if !ok {
				return ErrNotServing
			}
			_ = oldTp.Close() //nolint:errcheck
		default:
			mt.acceptCh <- tp
			return nil
		}
	}
}

// Dial dials a new underlying 'write' transport (and close/replace the old one).
func (mt *ManagedTransport) Dial(ctx context.Context) error {
	mt.dialMx.Lock()
	defer mt.dialMx.Unlock()

	if !mt.isServing() {
		return ErrNotServing
	}

	if mt.writeTp != nil {
		_ = mt.writeTp.Close() //nolint:errcheck
	}
	return mt.dial(ctx)
}

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
	mt.writeTp = tp
	return nil
}

// WritePacket writes a packet to the remote.
func (mt *ManagedTransport) WritePacket(ctx context.Context, rtID routing.RouteID, payload []byte) (err error) {
	mt.dialMx.Lock()
	defer mt.dialMx.Unlock()

	if !mt.isServing() {
		return ErrNotServing
	}

	if mt.writeTp == nil { // TODO: race condition
		if err := mt.dial(ctx); err != nil {
			return fmt.Errorf("failed to redial transport: %v", err)
		}
	}

	n, err := mt.writeTp.Write(routing.MakePacket(rtID, payload))
	if err != nil {
		if _, err := mt.dc.UpdateStatuses(context.Background(), &Status{ID: mt.Entry.ID, IsUp: false}); err != nil {
			mt.log.Warnf("Failed to change transport status: %s", err)
		}
		mt.writeTp = nil
		return err
	}
	if n > 0 {
		mt.logSent(uint64(len(payload)))
	}
	return nil
}

func (mt *ManagedTransport) latestReadTp() (Transport, error) {
	mt.acceptMx.RLock()
	defer mt.acceptMx.RUnlock()

	if mt.readTp != nil {
		return mt.readTp, nil
	}

	select {
	case <-mt.done:
		return nil, ErrNotServing

	case tp, ok := <-mt.acceptCh:
		if !ok {
			return nil, ErrNotServing
		}
		mt.readTp = tp
		return mt.readTp, nil
	}
}

// WARNING: Not thread safe.
func (mt *ManagedTransport) readPacket() (packet routing.Packet, err error) {
	tp, err := mt.latestReadTp()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && mt.isServing() {
			mt.acceptMx.RLock()
			mt.readTp = nil
			mt.acceptMx.RUnlock()
		}
	}()

	h := make(routing.Packet, 6)
	if _, err := io.ReadFull(tp, h); err != nil {
		return nil, err
	}

	p := make([]byte, h.Size())
	if _, err := io.ReadFull(tp, p); err != nil {
		return nil, err
	}
	packet = append(h, p...)
	mt.logRecv(uint64(len(p)))
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
