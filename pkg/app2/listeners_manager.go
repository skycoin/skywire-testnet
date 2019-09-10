package app2

import (
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

var (
	ErrPortAlreadyBound = errors.New("port is already bound")
	ErrNoListenerOnPort = errors.New("no listener on port")
	ErrWrongPID         = errors.New("wrong ProcID specified in the HS frame")
)

// listenersManager contains and manages all the instantiated listeners
type listenersManager struct {
	pid         ProcID
	pk          cipher.PubKey
	listeners   map[routing.Port]*listener
	mx          sync.RWMutex
	isListening int32
	logger      *logging.Logger
	doneCh      chan struct{}
	doneWg      sync.WaitGroup
}

func newListenersManager(l *logging.Logger, pid ProcID, pk cipher.PubKey) *listenersManager {
	return &listenersManager{
		pid:       pid,
		pk:        pk,
		listeners: make(map[routing.Port]*listener),
		logger:    l,
		doneCh:    make(chan struct{}),
	}
}

func (lm *listenersManager) close() {
	close(lm.doneCh)
	lm.doneWg.Wait()
}

func (lm *listenersManager) set(port routing.Port, l *listener) error {
	lm.mx.Lock()
	if v, ok := lm.listeners[port]; !ok || v != nil {
		lm.mx.Unlock()
		return ErrPortAlreadyBound
	}
	lm.listeners[port] = l
	lm.mx.Unlock()
	return nil
}

func (lm *listenersManager) reserveListener(port routing.Port) error {
	lm.mx.Lock()
	if _, ok := lm.listeners[port]; ok {
		lm.mx.Unlock()
		return ErrPortAlreadyBound
	}
	lm.listeners[port] = nil
	lm.mx.Unlock()
	return nil
}

func (lm *listenersManager) remove(port routing.Port) error {
	lm.mx.Lock()
	if _, ok := lm.listeners[port]; !ok {
		lm.mx.Unlock()
		return ErrNoListenerOnPort
	}
	delete(lm.listeners, port)
	lm.mx.Unlock()
	return nil
}

// addConn passes connection to the corresponding listener
func (lm *listenersManager) addConn(localPort routing.Port, remote routing.Addr, conn net.Conn) error {
	lm.mx.RLock()
	if _, ok := lm.listeners[localPort]; !ok {
		lm.mx.RUnlock()
		return ErrNoListenerOnPort
	}
	lm.listeners[localPort].addConn(&clientConn{
		remote: remote,
		Conn:   conn,
	})
	lm.mx.RUnlock()
	return nil
}

// listen accepts all new yamux streams from the server. We want to accept only
// `DmsgDial` frames here, thus all the other frames get rejected. `DmsgDial` frames
// are being distributed between the corresponding listeners with regards to their port
func (lm *listenersManager) listen(session *yamux.Session) {
	// this one should only start once
	if !atomic.CompareAndSwapInt32(&lm.isListening, 0, 1) {
		return
	}

	lm.doneWg.Add(1)

	go func() {
		defer lm.doneWg.Done()

		for {
			select {
			case <-lm.doneCh:
				return
			default:
				stream, err := session.Accept()
				if err != nil {
					lm.logger.WithError(err).Error("error accepting stream")
					return
				}

				hsFrame, err := readHSFrame(stream)
				if err != nil {
					lm.logger.WithError(err).Error("error reading HS frame")
					continue
				}

				if hsFrame.ProcID() != lm.pid {
					lm.logger.WithError(ErrWrongPID).Error("error listening for Dial")
				}

				if hsFrame.FrameType() != HSFrameTypeDMSGDial {
					lm.logger.WithError(ErrWrongHSFrameTypeReceived).Error("error listening for Dial")
					continue
				}

				// TODO: handle field get gracefully
				remotePort := routing.Port(binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen*2+HSFramePortLen:]))
				localPort := routing.Port(binary.BigEndian.Uint16(hsFrame[HSFrameHeaderLen+HSFramePKLen:]))

				var localPK cipher.PubKey
				copy(localPK[:], hsFrame[HSFrameHeaderLen:HSFrameHeaderLen+HSFramePKLen])

				err = lm.addConn(remotePort, routing.Addr{
					PubKey: localPK,
					Port:   localPort,
				}, stream)
				if err != nil {
					lm.logger.WithError(err).Error("failed to accept")
					continue
				}

				respHSFrame := NewHSFrameDMSGAccept(hsFrame.ProcID(), routing.Loop{
					Local: routing.Addr{
						PubKey: lm.pk,
						Port:   remotePort,
					},
					Remote: routing.Addr{
						PubKey: localPK,
						Port:   localPort,
					},
				})

				if _, err := stream.Write(respHSFrame); err != nil {
					lm.logger.WithError(err).Error("error responding with DmsgAccept")
					continue
				}
			}
		}
	}()
}
