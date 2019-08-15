package therealproxy

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"
	"golang.org/x/net/proxy"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			Logger.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

func TestProxy(t *testing.T) {
	srv, err := NewServer("")
	require.NoError(t, err)

	l, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)

	errChan := make(chan error)
	go func() {
		errChan <- srv.Serve(l)
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	require.NoError(t, err)

	client, err := NewClient(conn)
	require.NoError(t, err)

	errChan2 := make(chan error)
	go func() {
		errChan2 <- client.ListenAndServe(":10080")
	}()

	time.Sleep(100 * time.Millisecond)

	proxyDial, err := proxy.SOCKS5("tcp", ":10080", nil, proxy.Direct)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Hello, client")
		require.NoError(t, err)
	}))
	defer ts.Close()

	c := &http.Client{Transport: &http.Transport{Dial: proxyDial.Dial}}
	res, err := c.Get(ts.URL)
	require.NoError(t, err)

	msg, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, "Hello, client\n", string(msg))

	require.NoError(t, client.Close())
	require.NoError(t, srv.Close())

	<-errChan2
	<-errChan
}
