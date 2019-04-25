package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/util/pathutil"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

// EnvHostPK is the env provided to the hosted App for it to obtain the host's public key.
const EnvHostPK = "SW_APP_HOST"

const obtainMetaTimeout = time.Second * 10

// Errors associated with the app.Executor structure.
var (
	ErrAlreadyStarted = errors.New("app is already started")
	ErrAlreadyStopped = errors.New("app is already stopped")
)

// ObtainMeta runs '<app> sw-setup' to obtain app meta.
func ObtainMeta(hostPK cipher.PubKey, binLoc string) (*Meta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), obtainMetaTimeout)
	defer cancel()
	raw, err := exec.CommandContext(ctx, binLoc, setupCmdName).Output()
	l := log.
		WithField("_bin", binLoc).
		WithField("_stdout", string(raw))
	if err != nil {
		l.WithError(err).Error("failed to obtain app meta")
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(raw, &meta); err != nil {
		l.WithError(err).Error("failed to obtain app meta")
		return nil, err
	}
	if meta.ProtocolVersion != ProtocolVersion {
		err := fmt.Errorf("app uses protocol version (%s) when only (%s) is supported", meta.ProtocolVersion, ProtocolVersion)
		l.WithError(err).Error("failed to obtain app meta")
		return nil, err
	}
	meta.Host = hostPK
	log.Printf("obtained app meta: %s", string(raw[:len(raw)-1]))
	return &meta, err
}

// ExecConfig configures the executor.
type ExecConfig struct {
	HostPK  cipher.PubKey `json:"-"`
	HostSK  cipher.SecKey `json:"-"`
	WorkDir string        `json:"work_dir"`
	BinLoc  string        `json:"bin_loc"`
	Args    []string      `json:"args"`
}

// Process checks all fields.
func (ec *ExecConfig) Process() error {
	// Check and process binary file.
	binLoc, err := filepath.Abs(ec.BinLoc)
	if err != nil {
		return fmt.Errorf("invalid binary file: %s", err)
	}
	if _, err := os.Stat(binLoc); os.IsNotExist(err) {
		return fmt.Errorf("invalid binary file: %s", err)
	}
	ec.BinLoc = binLoc

	// Check and process working directory.
	wkDir, err := pathutil.EnsureDir(ec.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to init working directory '%s': %s", ec.WorkDir, err)
	}
	ec.WorkDir = wkDir

	return nil
}

// AppName obtains the app's name.
func (ec *ExecConfig) AppName() string {
	return filepath.Base(ec.BinLoc)
}

// Executor is used by the App's host to run, stop and communicate with the App.
// Regarding thread-safety;
// - The .Start() and .Stop() members should always be executed on the same thread/go-routine.
// - The .Call() and .CallUI() members are thread-safe, as long as one .Start() command is run prior.
type Executor struct {
	c     *ExecConfig
	m     *Meta
	proc  *os.Process
	dataP *appnet.Protocol // Data Protocol: used for establishing and sending/receiving packets within the Skywire Network.
	ctrlP *appnet.Protocol // Control Protocol: used for proxying the App's interface (e.g. for access by the Manager Node).
	wg    *sync.WaitGroup  // Also used to determine whether app is running or not (nil == not running).
	log   *logging.Logger
}

// NewExecutor creates a structure that is used by an App's host.
func NewExecutor(l *logging.Logger, m *Meta, c *ExecConfig) (*Executor, error) {
	if err := c.Process(); err != nil {
		return nil, err
	}
	return &Executor{
		c:   c,
		m:   m,
		log: l,
	}, nil
}

// Run executes the App and serves the 2 piped connections.
// When the App quits, the <-chan struct{} output will be notified.
func (h *Executor) Run(dataHM, ctrlHM appnet.HandlerMap) (<-chan struct{}, error) {
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
	cmd := exec.Command(h.c.BinLoc, h.c.Args...) //nolint:gosec
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", EnvHostPK, h.c.HostPK.String()))
	cmd.ExtraFiles = append(appConn.Files(), uiAppConn.Files()...)
	cmd.Stdout = h.log.WithField("_src", "stdout").Writer()
	cmd.Stderr = h.log.WithField("_src", "stderr").Writer()
	cmd.Dir = h.c.WorkDir
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("run_app: %s", err)
	}
	h.proc = cmd.Process
	h.dataP = appnet.NewProtocol(hostConn)
	h.ctrlP = appnet.NewProtocol(uiHostConn)
	h.wg = new(sync.WaitGroup)
	h.wg.Add(3)
	doneCh := make(chan struct{}, 1)
	go func() {
		if err := h.dataP.Serve(dataHM); err != nil {
			h.log.Errorf("proto exited with err: %s", err.Error())
		}
		h.wg.Done()
	}()
	go func() {
		if err := h.ctrlP.Serve(ctrlHM); err != nil {
			h.log.Errorf("ui proto exited with err: %s", err.Error())
		}
		h.wg.Done()
	}()
	go func() {
		if err := cmd.Wait(); err != nil {
			h.log.Warnf("cmd exited with err: %s", err.Error())
		}
		_ = h.dataP.Close() //nolint:errcheck
		_ = h.ctrlP.Close() //nolint:errcheck
		h.wg.Done()
		doneCh <- struct{}{}
		close(doneCh)
	}()
	return doneCh, nil
}

// Stop sends a SIGTERM signal to the app, and waits for the app to quit.
func (h *Executor) Stop() error {
	if h.wg == nil {
		return ErrAlreadyStopped
	}
	err := h.proc.Signal(syscall.SIGTERM)
	h.wg.Wait()
	h.wg = nil
	return err
}

// Conf obtains the internal config.
func (h *Executor) Conf() *ExecConfig { return h.c }

// Meta returns the hosted app's meta data.
func (h *Executor) Meta() *Meta { return h.m }

// Call sends a command to the App via the regular piped connection.
func (h *Executor) Call(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return h.dataP.Call(t, reqData)
}

// CallUI sends a command to the App via the ui piped connection.
func (h *Executor) CallUI(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return h.ctrlP.Call(t, reqData)
}
