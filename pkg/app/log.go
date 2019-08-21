package app

import (
	"bytes"
	"fmt"
	"time"

	"encoding/binary"
	"go.etcd.io/bbolt"
)

// LogStore stores logs from apps, for later consumption from the hypervisor
type LogStore interface {

	// Store saves given log in db
	Store(t time.Time, s string) error

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

// Store implements LogStore
func (l *boltDBappLogs) Store(t time.Time, s string) error {
	parsedTime := make([]byte, 16)
	binary.BigEndian.PutUint64(parsedTime, uint64(t.UnixNano()))

	return l.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(l.bucket)
		return b.Put(parsedTime, []byte(s))
	})
}

// LogSince implements LogStore
func (l *boltDBappLogs) LogsSince(t time.Time) ([]string, error) {
	logs := make([]string, 0)

	err := l.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(l.bucket)
		parsedTime := make([]byte, 16)
		binary.BigEndian.PutUint64(parsedTime, uint64(t.UnixNano()))
		c := b.Cursor()

		v := b.Get(parsedTime)
		if v == nil {
			iterateFromBeginning(c, parsedTime, &logs)
			return nil
		}
		if k, _ := c.Seek(parsedTime); k != nil {
			iterateFromKey(c, &logs)
		}

		return nil
	})

	return logs, err
}

func iterateFromKey(c *bbolt.Cursor, logs *[]string) {
	for k, v := c.Next(); k != nil; k, v = c.Next() {
		*logs = append(*logs, fmt.Sprintf("%s-%s", bytesToTime(k).UTC().Format(time.RFC3339Nano), string(v)))
	}
}

func iterateFromBeginning(c *bbolt.Cursor, parsedTime []byte, logs *[]string) {
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if bytes.Compare(k, parsedTime) < 0 {
			continue
		}

		*logs = append(*logs, fmt.Sprintf("%s-%s", bytesToTime(k).UTC().Format(time.RFC3339Nano), string(v)))
	}
}

func bytesToTime(b []byte) time.Time {
	return time.Unix(int64(binary.BigEndian.Uint64(b)), 0)
}
