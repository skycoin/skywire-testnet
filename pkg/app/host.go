package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Host is used by the App's host to run, stop and communicate with the App.
// Regarding thread-safety;
// - The .Start() and .Stop() members should always be executed on the same thread/go-routine.
// - The .Call() and .CallUI() members are thread-safe, as long as one .Start() command is run prior.
type Host struct {
	Meta
	binLoc  string
	workDir string
	args    []string
	log     *logrus.Entry

	proc    *os.Process
	proto   *appnet.Protocol // used for establishing and sending/receiving packets within the Skywire Network.
	uiProto *appnet.Protocol // used for proxying the user interface of the Skywire App (e.g. for access by the Manager Node).

	wg   *sync.WaitGroup
}

// Errors associated with the app.Host structure.
var (
	ErrAlreadyStarted = errors.New("app is already started")
	ErrAlreadyStopped = errors.New("app is already stopped")
)

// NewHost creates a structure that is used by an App's host.
func NewHost(hostPK cipher.PubKey, workDir, binLoc string, args []string) (*Host, error) {
	obtainMeta := func() (Meta, error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		cmd := exec.CommandContext(ctx, binLoc, setupCmdName)
		raw, err := cmd.Output()
		if err != nil {
			return Meta{}, err
		}
		var conf Meta
		err = json.Unmarshal(raw, &conf)
		return conf, err
	}
	processMeta := func(meta *Meta) error {
		if binName := filepath.Base(binLoc); meta.AppName != binName {
			return fmt.Errorf("configured app name (%s) does not match bin name (%s)",
				meta.AppName, binName)
		}
		if meta.ProtocolVersion != protocolVersion {
			return fmt.Errorf("app uses protocol version (%s) when only (%s) is supported",
				meta.ProtocolVersion, protocolVersion)
		}
		meta.Host = hostPK
		return nil
	}
	meta, err := obtainMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain app config: %s", err.Error())
	}
	if err := processMeta(&meta); err != nil {
		return nil, fmt.Errorf("unable to start app: %s", err.Error())
	}
	return &Host{
		Meta:    meta,
		binLoc:  binLoc,
		workDir: workDir,
		args:    args,
		log:     log.WithFields(logrus.Fields{"_app": meta.AppName}),
	}, nil
}

// Start executes the App and serves the 2 piped connections.
// When the App quits, the <-chan struct{} output will be notified.
func (h *Host) Start(handler, uiHandler appnet.HandlerMap) (<-chan struct{}, error) {
	if h.wg != nil {
		return nil, ErrAlreadyStarted
	}
	hostConn, appConn, err := appnet.OpenPipeConn()
	if err != nil {
		return nil, fmt.Errorf("failed to open pipe: %s", err)
	}
	uiHostConn, uiAppConn, err := appnet.OpenPipeConn()
	if err != nil {
		return nil, fmt.Errorf("failed to open ui pipe: %s", err)
	}
	cmd := exec.Command(h.binLoc, append([]string{h.Host.String()}, h.args...)...)
	cmd.ExtraFiles = append(appConn.Files(), uiAppConn.Files()...)
	cmd.Stdout = h.log.WithField("_src", "stdout").Writer()
	cmd.Stderr = h.log.WithField("_src", "stderr").Writer()
	cmd.Dir = h.workDir
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("run_app: %s", err)
	}
	h.proc = cmd.Process
	h.proto = appnet.NewProtocol(hostConn)
	h.uiProto = appnet.NewProtocol(uiHostConn)
	h.wg = new(sync.WaitGroup)
	h.wg.Add(3)
	doneCh := make(chan struct{}, 1)
	go func() {
		if err := h.proto.Serve(handler); err != nil {
			h.log.Errorf("proto exited with err: %s", err.Error())
		}
		h.wg.Done()
	}()
	go func() {
		if err := h.uiProto.Serve(uiHandler); err != nil {
			h.log.Errorf("ui proto exited with err: %s", err.Error())
		}
		h.wg.Done()
	}()
	go func() {
		if err := cmd.Wait(); err != nil {
			h.log.Warnf("cmd exited with err: %s", err.Error())
		}
		_ = h.proto.Close()   //nolint:errcheck
		_ = h.uiProto.Close() //nolint:errcheck
		h.wg.Done()
		doneCh <- struct{}{}
		close(doneCh)
	}()
	return doneCh, nil
}

// Call sends a command to the App via the regular piped connection.
func (h *Host) Call(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return h.proto.Call(t, reqData)
}

// CallUI sends a command to the App via the ui piped connection.
func (h *Host) CallUI(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return h.uiProto.Call(t, reqData)
}

// Stop sends a SIGTERM signal to the app, and waits for the app to quit.
func (h *Host) Stop() error {
	if h.wg == nil {
		return ErrAlreadyStopped
	}
	err := h.proc.Signal(syscall.SIGTERM)
	h.wg.Wait()
	h.wg = nil
	return err
}
