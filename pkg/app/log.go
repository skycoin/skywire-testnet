package app

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

type LogStore interface {
}

type boltDBappLogs struct {
	db     *bbolt.DB
	bucket []byte
}

func newBoltDB(path, appName string) (LogStore, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	b := []byte(appName)
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(b); err != nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &boltDBappLogs{db, b}, nil
}

func (l *boltDBappLogs) LogsSince(time time.Time) ([]string, error) {
	err := l.db.View()
}
