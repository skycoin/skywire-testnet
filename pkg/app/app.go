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
	configCmdName   = "sw-config"
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
	Host            cipher.PubKey `json:"host"`
}

var (
	_meta     Meta
	_proto    *appnet.Protocol
	_acceptCh chan LoopMeta
	_doneCh   chan struct{}
	_loops    map[LoopMeta]io.ReadWriteCloser
	_mu       sync.RWMutex
)

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

	// If command is of format: "<exec> sw-config", print json-encoded appnet.Config, otherwise, serve app.
	if os.Args[1] == configCmdName {
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

		_proto = appnet.NewProtocol(pipeConn)
		_acceptCh = make(chan LoopMeta)
		_doneCh = make(chan struct{})
		_loops = make(map[LoopMeta]io.ReadWriteCloser)

		// used to obtain host's public key.
		pkCh := make(chan cipher.PubKey, 1)

		// Serve the connection between host and this App.
		go func() {
			if err := serveHostConn(); err != nil {
				log.Fatalf("Error: %s", err.Error())
			}
		}()

		// obtain the host's public key before finishing setup.
		_meta.Host = <-pkCh
		close(pkCh)
	}
}

// this serves the connection between the host and this App.
func serveHostConn() error {

	handleConfirmLoop := func(lm LoopMeta) error {
		_mu.Lock()
		_, ok := _loops[lm]
		_mu.Unlock()
		if !ok {
			return errors.New("loop is already created")
		}
		select {
		case _acceptCh <- lm:
		default:
		}
		return nil
	}

	handleCloseLoop := func(lm LoopMeta) error {
		_mu.Lock()
		conn, ok := _loops[lm]
		_mu.Unlock()
		if !ok {
			return nil
		}
		delete(_loops, lm)
		return conn.Close()
	}

	handleData := func(df DataFrame) error {
		_mu.Lock()
		conn, ok := _loops[df.Meta]
		_mu.Unlock()
		if !ok {
			return fmt.Errorf("received packet is directed at non-existent loop: %v", df.Meta)
		}
		_, err := conn.Write(df.Data)
		return err
	}

	return _proto.Serve(func(t appnet.FrameType, bytes []byte) (interface{}, error) {
		switch t {
		case appnet.FrameConfirmLoop:
			var lm LoopMeta
			if err := json.Unmarshal(bytes, &lm); err != nil {
				return nil, err
			}
			return nil, handleConfirmLoop(lm)

		case appnet.FrameCloseLoop:
			var lm LoopMeta
			if err := json.Unmarshal(bytes, &lm); err != nil {
				return nil, err
			}
			return nil, handleCloseLoop(lm)

		case appnet.FrameData:
			var df DataFrame
			if err := json.Unmarshal(bytes, &df); err != nil {
				return nil, err
			}
			return nil, handleData(df)

		default:
			return nil, errors.New("unexpected frame type")
		}
	})
}

// Close closes the app.
func Close() error {
	select {
	case <-_doneCh:
		return ErrAppClosed
	default:
	}

	_mu.Lock()
	for addr, l := range _loops {
		_ = _proto.Send(appnet.FrameCloseLoop, &addr, nil) //nolint:errcheck
		_ = l.Close()
	}
	_mu.Unlock()

	return _proto.Close()
}

// Info obtains meta information of the App.
func Info() Meta { return _meta }

// Accept awaits for incoming loop confirmation request from a Node and returns net.Conn for received loop.
func Accept() (net.Conn, error) {
	select {
	case <-_doneCh:
		return nil, ErrAppClosed
	case lm := <-_acceptCh:
		return newLoopConn(lm), nil
	}
}

// Dial sends create loop request to a Node and returns net.Conn for created loop.
func Dial(remoteAddr LoopAddr) (net.Conn, error) {
	select {
	case <-_doneCh:
		return nil, ErrAppClosed
	default:
	}

	var localAddr LoopAddr
	if err := _proto.Send(appnet.FrameCreateLoop, remoteAddr, localAddr); err != nil {
		return nil, err
	}
	lm := LoopMeta{Local: localAddr, Remote: remoteAddr}
	return newLoopConn(lm), nil
}
