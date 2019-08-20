package app

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"go.etcd.io/bbolt"
)

// LogStore stores logs from apps, for later consumption from the hypervisor
type LogStore interface {

	// Store saves given log in db
	Store(t time.Time, string) error

	// LogSince returns the logs since given timestamp. For optimal performance,
	// the timestamp should exist in the store (you can get it from previous logs),
	// otherwise the DB will be sequentially iterated until finding entries older than given timestamp
	LogsSince(t time.Time) ([]string, error)
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

// LogSince implements LogStore
func (l *boltDBappLogs) LogsSince(t time.Time) ([]string, error) {
	logs := make([]string, 0)

	err := l.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(l.bucket)

		c := b.Cursor()
		if k, _ := c.Seek([]byte(t.Format(time.RFC3339))); k != nil {
			iterateFromKey(c, logs)
		} else {
			iterateFromBeginning(c, t, logs)
		}

		return nil
	})

	return logs, err
}

func iterateFromKey(c *bbolt.Cursor, logs []string) {
	for k, v := c.Next(); k != nil; k, v = c.Next() {
		logs = append(logs, fmt.Sprintf("%s-%s", string(k), string(v)))
	}
}

func iterateFromBeginning(c *bbolt.Cursor, t time.Time, logs []string) {
	parsedT := []byte(t.UTC().Format(time.RFC3339))

	for k, v := c.First(); k != nil; k, v = c.Next() {
		if bytes.Compare(parsedT, k) < 0 {
			continue
		}

		logs = append(logs, fmt.Sprintf("%s-%s", string(k), string(v)))
	}
}
