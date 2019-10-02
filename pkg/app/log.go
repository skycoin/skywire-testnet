package app

import (
	"io"
	"os"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"
)

// NewLogger returns a logger which persists app logs. This logger should be passed down
// for use on any other function used by the app. It's configured from an additional app argument.
// It modifies os.Args stripping from it such value. Should be called before using os.Args inside the app
func NewLogger(appName string) *logging.MasterLogger {
	db, err := newBoltDB(os.Args[1], appName)
	if err != nil {
		panic(err)
	}

	l := newAppLogger()
	l.SetOutput(io.MultiWriter(l.Out, db))
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)

	return l
}

// TimestampFromLog is an utility function for retrieving the timestamp from a log. This function should be modified
// if the time layout is changed
func TimestampFromLog(log string) string {
	return log[1:36]
}

func (app *App) newPersistentLogger(path string) (*logging.MasterLogger, LogStore, error) {
	db, err := newBoltDB(path, app.config.AppName)
	if err != nil {
		return nil, nil, err
	}

	l := newAppLogger()
	l.SetOutput(io.MultiWriter(l.Out, db))

	return l, db, nil
}

func newAppLogger() *logging.MasterLogger {
	l := logging.NewMasterLogger()
	l.Logger.Formatter.(*logging.TextFormatter).TimestampFormat = time.RFC3339Nano
	return l
}
