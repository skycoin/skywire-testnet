package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func randMeta(i int, hPK cipher.PubKey) Meta {
	return Meta{
		AppName:         fmt.Sprintf("app_%d", i),
		AppVersion:      "0.0.1",
		ProtocolVersion: ProtocolVersion,
		Host:            hPK,
	}
}

func genMockApp(path string, m Meta) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(0777))
	if err != nil {
		return err
	}
	defer func() { err = f.Close() }()

	jm, err := json.Marshal(m)
	if err != nil {
		return err
	}

	template := `#!/bin/bash
PK="${%s}"
if [[ $# -eq 1 && $1 = '%s' ]]; then echo '%s'; exit 0
elif [[ -n "${PK}" ]]; then echo "host: ${PK}"; while [ 1 ]; do test $? -gt 128 && exit 0; done
else exit 1
fi`
	_, err = fmt.Fprintf(f, template, EnvHostPK, setupCmdName, string(jm))
	return err
}

func TestNewExecutor(t *testing.T) {
	const runs = 20

	wkDir, err := os.Getwd()
	require.NoError(t, err)

	// temp dir for mock app binaries.
	appDir, err := ioutil.TempDir(os.TempDir(), "sw_app")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.RemoveAll(appDir)) }()

	for i := 0; i < runs; i++ {
		var (
			pk, sk = cipher.GenerateKeyPair()
			m      = randMeta(i, pk)
			binLoc = filepath.Join(appDir, m.AppName)
		)

		// Generate a mock app binary.
		require.NoError(t, genMockApp(binLoc, m))

		// Obtain meta.
		meta, err := ObtainMeta(pk, binLoc)
		require.NoError(t, err)
		require.Equal(t, m, *meta)

		// Create app host and check obtained AppMeta.
		host, err := NewExecutor(
			logging.MustGetLogger("executor"),
			meta,
			&ExecConfig{
				HostPK:  pk,
				HostSK:  sk,
				WorkDir: wkDir,
				BinLoc:  binLoc,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, m, *host.Meta())

		// It is expected that a 'Host' struct is reusable.
		// We will start and stop the App via the 'Host' 3 times.
		for j := 0; j < 3; j++ {

			// Start app from host.
			done, err := host.Run(nil, nil)
			assert.NoError(t, err)

			// This should fail as app has already started.
			_, err = host.Run(nil, nil)
			assert.EqualError(t, err, ErrAlreadyStarted.Error())

			time.Sleep(time.Millisecond * 5)

			// Stop app from host.
			assert.NoError(t, host.Stop())
			<-done

			// This should fail as app has already stopped.
			assert.EqualError(t, host.Stop(), ErrAlreadyStopped.Error())
		}
	}
}

// TODO(evanlinjin): write tests for .Call() and .CallUI()
