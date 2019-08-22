package app

import (
	"github.com/skycoin/skycoin/src/util/logging"
	"io"
	"time"
)

// NewLogger is like (a *App) LoggerFromArguments but with appName as parameter, instead of
// getting it from app config
func NewLogger(appName string, args []string) (*logging.MasterLogger, []string) {
	db, err := newBoltDB(args[1], appName)
	if err != nil {
		panic(err)
	}

	l := newAppLogger()
	l.SetOutput(io.MultiWriter(l.Out, db))

	return l, append([]string{args[0]}, args[2:]...)
}

// LoggerFromArguments returns a logger which persists app logs. This logger should be passed down
// for use on any other function used by the app. It's configured from an additional app argument.
// It also returns the args list with such argument stripped from it, for convenience
func (a *App) LoggerFromArguments(args []string) (*logging.MasterLogger, []string) {
	l, _, err := a.newPersistentLogger(args[1])
	if err != nil {
		panic(err)
	}

	return l, append([]string{args[0]}, args[2:]...)
}

func (a *App) newPersistentLogger(path string) (*logging.MasterLogger, LogStore, error) {
	db, err := newBoltDB(path, a.config.AppName)
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
