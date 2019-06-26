package ioutil

import (
	"log"
	"net"
	"os"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

func TestLenReadWriter(t *testing.T) {
	in, out := net.Pipe()
	rwIn := NewLenReadWriter(in)
	rwOut := NewLenReadWriter(out)

	errCh := make(chan error)
	go func() {
		_, err := rwIn.Write([]byte("foo"))
		errCh <- err
	}()

	buf := make([]byte, 2)
	n, err := rwOut.Read(buf)
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte("fo"), buf)

	buf = make([]byte, 2)
	n, err = rwOut.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte("o"), buf[:n])

	go func() {
		_, err := rwIn.Write([]byte("foo"))
		errCh <- err
	}()

	packet, err := rwOut.ReadPacket()
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, []byte("foo"), packet)

	go func() {
		_, err := rwOut.ReadPacket()
		errCh <- err
	}()

	n, err = rwIn.Write([]byte("bar"))
	require.NoError(t, err)
	require.NoError(t, <-errCh)
	assert.Equal(t, 3, n)
}
