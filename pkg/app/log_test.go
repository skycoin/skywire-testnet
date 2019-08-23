package app

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewLogger tests that after the new logger is created logs with it are persisted into storage
func TestNewLogger(t *testing.T) {
	p, err := ioutil.TempFile("", "test-db")
	require.NoError(t, err)

	defer os.Remove(p.Name())

	a := &App{
		config: Config{
			AppName: "foo",
		},
	}

	l, _, err := a.newPersistentLogger(p.Name())
	require.NoError(t, err)

	dbl, err := newBoltDB(p.Name(), a.config.AppName)
	require.NoError(t, err)

	l.Info("bar")

	beggining := time.Unix(0, 0)
	res, err := dbl.(*boltDBappLogs).LogsSince(beggining)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Contains(t, res[0], "bar")
}
