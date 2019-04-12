package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/cipher"
)

// Host is used by the App's host to run, stop and communicate with the App.
type Host struct {
	Meta
	Cmd    *exec.Cmd
	Conn   net.Conn // used for establishing and sending/receiving packets within the Skywire Network.
	UIConn net.Conn // used for proxying the user interface of the Skywire App (e.g. for access by the Manager Node).
}

// NewHost creates a structure that is used by an App's host.
func NewHost(hostPK cipher.PubKey, binPath string, args []string) (*Host, error) {

	obtainMeta := func() (Meta, error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		cmd := exec.CommandContext(ctx, binPath, setupCmdName)
		raw, err := cmd.Output()
		if err != nil {
			return Meta{}, err
		}
		var conf Meta
		err = json.Unmarshal(raw, &conf)
		return conf, err
	}

	checkMeta := func(conf Meta) error {
		if binName := filepath.Base(binPath); conf.AppName != binName {
			return fmt.Errorf("configured app name (%s) does not match bin name (%s)",
				conf.AppName, binName)
		}
		if conf.ProtocolVersion != protocolVersion {
			return fmt.Errorf("app uses protocol version (%s) when only (%s) is supported",
				conf.ProtocolVersion, protocolVersion)
		}
		return nil
	}

	meta, err := obtainMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain app config: %s", err.Error())
	}

	if err := checkMeta(meta); err != nil {
		return nil, fmt.Errorf("unable to start app: %s", err.Error())
	}

	hostConn, appConn, err := appnet.OpenPipeConn()
	if err != nil {
		return nil, fmt.Errorf("failed to open piped connection: %s", err)
	}

	cmd := exec.Command(binPath, append([]string{hostPK.String()}, args...)...)
	cmd.ExtraFiles = appConn.Files()
	return &Host{Meta: meta, Cmd: cmd, Conn: hostConn}, nil
}
