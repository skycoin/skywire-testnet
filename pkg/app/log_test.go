package app

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/require"
)

func TestWriteLog(t *testing.T) {
	r, w := io.Pipe()

	l := logging.NewMasterLogger()
	l.SetOutput(w)
	l.Logger.Formatter.(*logging.TextFormatter).TimestampFormat = time.RFC3339Nano
	c := make(chan []byte)

	go func() {
		b := make([]byte, 51)
		r.Read(b)
		c <- b
	}()
	l.Println("foo")

	res := <-c
	ti := res[1:36]

	pt, err := time.Parse(time.RFC3339Nano, string(ti))
	if err != nil {
		t.Fail()
	}

	fmt.Println("t in unix nano", pt.UnixNano())
	fmt.Printf("%#v", string(res))
}

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

	// here we parse the layout itself since it's a date from 2006, so it is earlier than any other logs produced now.
	// The last 5 characters are extracted since otherwise it cannot be parsed
	beggining, err := time.Parse(time.RFC3339Nano, time.RFC3339Nano[:len(time.RFC3339Nano)-5])
	require.NoError(t, err)
	res, err := dbl.(*boltDBappLogs).LogsSince(beggining)
	require.NoError(t, err)
	require.Len(t, res, 1)
	fmt.Println("from db: ", res[0])
	fmt.Println(time.Now().Format(time.RFC3339Nano))
	require.Contains(t, res[0], "bar")
}
