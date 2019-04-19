package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func randMeta(i int) Meta {
	return Meta{
		AppName:         fmt.Sprintf("app_%d", i),
		AppVersion:      "0.0.1",
		ProtocolVersion: protocolVersion,
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
elif [[ -n $1 ]]; then exit 0
else exit 1
fi`
	_, err = fmt.Fprintf(f, template, setupCmdName, string(jm))
	return err
}

func TestNewHost(t *testing.T) {
	const appCount = 20

	appDir, err := ioutil.TempDir(os.TempDir(), "sw_app")
	require.NoError(t, err)
	defer func() { require.NoError(t, os.RemoveAll(appDir)) }()

	for i := 0; i < appCount; i++ {
		var (
			pk, _  = cipher.GenerateKeyPair()
			m      = randMeta(i)
			binLoc = filepath.Join(appDir, m.AppName)
		)

		require.NoError(t, genMockApp(binLoc, m))

		host, err := NewHost(pk, binLoc, nil)
		require.NoError(t, err)
		assert.Equal(t, m, host.Meta)
	}
}
