/*
Package app implements app to node communication interface.
*/
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	protocolVersion = "0.0.1"
	setupCmdName    = "sw-setup"
)

var (
	// ErrAppClosed occurs when an action is executed when the app is closed.
	ErrAppClosed = errors.New("app closed")

	// for logging.
	log = logging.MustGetLogger("app")
)

// Meta contains meta data for the app.
type Meta struct {
	AppName         string        `json:"app_name"`
	AppVersion      string        `json:"app_version"`
	ProtocolVersion string        `json:"protocol_version"`
	Host            cipher.PubKey `json:"-"`
}

var (
	_meta      Meta
	_proto     *appnet.Protocol
	_acceptCh  chan LoopMeta
	_loopPipes map[LoopMeta]io.ReadWriteCloser
	_mu        = new(sync.RWMutex)
)

func loopPipe(lm LoopMeta) (io.ReadWriteCloser, bool) {
	_mu.RLock()
	lp, ok := _loopPipes[lm]
	_mu.RUnlock()
	return lp, ok
}

func setLoopPipe(lm LoopMeta, rw io.ReadWriteCloser) error {
	_mu.Lock()
	if _, ok := _loopPipes[lm]; ok {
		_mu.Unlock()
		return fmt.Errorf("already handling loop '%s'", lm.String())
	}
	_loopPipes[lm] = rw
	_mu.Unlock()
	return nil
}

func rmLoopPipe(lm LoopMeta) {
	_mu.Lock()
	if _, ok := _loopPipes[lm]; ok {
		if _, err := _proto.Call(appnet.FrameCloseLoop, lm.Encode()); err != nil && err != io.ErrClosedPipe {
			log.Warnf("failed to send 'CloseLoop': %s", err.Error())
		}
		delete(_loopPipes, lm)
	}
	_mu.Unlock()
}

// Setup initiates the app. Module will hang if Setup() is not run.
func Setup(appName, appVersion string) {
	_meta = Meta{
		AppName:         appName,
		AppVersion:      appVersion,
		ProtocolVersion: protocolVersion,
	}

	if len(os.Args) < 2 {
		log.Fatal("App expects at least 2 arguments")
	}

	// If command is of format: "<app> sw-setup", print json-encoded Meta, otherwise, serve app.
	if os.Args[1] == setupCmdName {
		if appName != os.Args[0] {
			log.Fatalf("Registered name '%s' does not match executable name '%s'.", appName, os.Args[0])
		}
		if err := json.NewEncoder(os.Stdout).Encode(_meta); err != nil {
			log.Fatalf("Failed to write to stdout: %s", err.Error())
		}
		os.Exit(0)

	} else {
		if err := _meta.Host.Set(os.Args[1]); err != nil {
			log.Fatalf("host provided invalid public key: %s", err.Error())
		}

		pipeConn, err := appnet.NewPipeConn(appnet.DefaultIn, appnet.DefaultOut)
		if err != nil {
			log.Fatalf("Setup failed to open pipe: %s", err.Error())
		}

		_mu.Lock()
		_proto = appnet.NewProtocol(pipeConn)
		_acceptCh = make(chan LoopMeta)
		_loopPipes = make(map[LoopMeta]io.ReadWriteCloser)
		_mu.Unlock()

		// Serve the connection between host and this App.
		go func() {
			if err := serveHostConn(); err != nil {
				log.Fatalf("Error: %s", err.Error())
			}
		}()
	}
}

// this serves the connection between the host and this App.
func serveHostConn() error {
	return _proto.Serve(appnet.HandlerMap{
		appnet.FrameConfirmLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var lm LoopMeta
			if err := lm.Decode(b); err != nil {
				return nil, err
			}
			select {
			case _acceptCh <- lm:
			default:
			}
			return nil, nil
		},
		appnet.FrameCloseLoop: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var lm LoopMeta
			if err := lm.Decode(b); err != nil {
				return nil, err
			}
			conn, ok := loopPipe(lm)
			if !ok {
				return nil, nil
			}
			delete(_loopPipes, lm)
			return nil, conn.Close()
		},
		appnet.FrameData: func(_ *appnet.Protocol, b []byte) ([]byte, error) {
			var df DataFrame
			if err := df.Decode(b); err != nil {
				return nil, err
			}
			conn, ok := loopPipe(df.Meta)
			if !ok {
				return nil, fmt.Errorf("received packet is directed at non-existent loop: %v", df.Meta)
			}
			_, err := conn.Write(df.Data)
			return nil, err
		},
	})
}

// Close closes the app.
func Close() error {
	_mu.Lock()
	for lm, l := range _loopPipes {
		_, _ = _proto.Call(appnet.FrameCloseLoop, lm.Encode()) //nolint:errcheck
		_ = l.Close()
	}
	_mu.Unlock()

	if err := _proto.Close(); err != nil {
		return err
	}
	close(_acceptCh)
	return nil
}

// Info obtains meta information of the App.
func Info() Meta { return _meta }

// Accept awaits for incoming loop confirmation request from a Node and returns net.Conn for received loop.
func Accept() (net.Conn, error) {
	select {
	case lm, ok := <-_acceptCh:
		if !ok {
			return nil, ErrAppClosed
		}
		return setAndServeLoop(lm)
	}
}

// DialFunc is the method for dialing operations.
type DialFunc func(remoteAddr LoopAddr) (net.Conn, error)

// Dial sends create loop request to a Node and returns net.Conn for created loop.
func Dial(remoteAddr LoopAddr) (net.Conn, error) {
	lmRaw, err := _proto.Call(appnet.FrameCreateLoop, remoteAddr.Encode())
	if err != nil {
		return nil, err
	}
	var lm LoopMeta
	if err := lm.Decode(lmRaw); err != nil {
		return nil, err
	}
	if lm.Remote != remoteAddr {
		log.Fatalf("Dial: Received unexpected loop meta response from App host: %s", lm.String())
	}
	fmt.Println("Dial: preparing to serve loop:", lm)
	return setAndServeLoop(lm)
}

// TODO(evanlinjin): The following implementations of net.Listener is temporary.
type Listener struct{}

func (l *Listener) Accept() (net.Conn, error) { return Accept() }
func (l *Listener) Close() error              { return Close() }
func (l *Listener) Addr() net.Addr            { return &LoopAddr{PubKey: _meta.Host, Port: 0} }
