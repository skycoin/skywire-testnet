package app

import (
	"encoding/json"
	"fmt"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func randMeta(i int, hPK cipher.PubKey) Meta {
	return Meta{
		AppName:         fmt.Sprintf("app_%d", i),
		AppVersion:      "0.0.1",
		ProtocolVersion: protocolVersion,
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
if [[ $# -ne 1 ]]; then exit 1
elif [[ $1 = '%s' ]]; then echo '%s'
elif [[ -n $1 ]]; then echo 'success!'
else exit 1
fi
exit 0`
	_, err = fmt.Fprintf(f, template, setupCmdName, string(jm))
	return err
}

func TestNewHost(t *testing.T) {
	const appCount = 20

	wkDir, err := os.Getwd()
	require.NoError(t, err)

	// temp dir for mock app binaries.
	appDir, err := ioutil.TempDir(os.TempDir(), "sw_app")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.RemoveAll(appDir)) }()

	for i := 0; i < appCount; i++ {
		var (
			pk, _  = cipher.GenerateKeyPair()
			m      = randMeta(i, pk)
			binLoc = filepath.Join(appDir, m.AppName)
		)

		// Generate a mock app binary.
		require.NoError(t, genMockApp(binLoc, m))

		// Create app host and check obtained AppMeta.
		host, err := NewHost(pk, wkDir, binLoc, nil)
		require.NoError(t, err)
		assert.Equal(t, m, host.Meta)

		// It is expected that a 'Host' struct is reusable.
		// We will start and stop the App via the 'Host' 3 times.
		for j := 0; j < 3; j++ {

			// Start app from host.
			done, err := host.Start(nil, nil)
			assert.NoError(t, err)

			// This should fail as app has already started.
			_, err = host.Start(nil, nil)
			assert.EqualError(t, err, ErrAlreadyStarted.Error())

			// Stop app from host.
			assert.NoError(t, host.Stop())
			select {
			case <-done:
			default:
				t.Error("unexpected empty done chan")
			}

			// This should fail as app has already stopped.
			assert.EqualError(t, host.Stop(), ErrAlreadyStopped.Error())
		}
	}
}

// TODO(evanlinjin): write tests for .Call() and .CallUI()