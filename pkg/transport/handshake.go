package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/skycoin/dmsg/cipher"
)

type settlementHandshake func(tm *Manager, tr Transport) (*Entry, error)

func (handshake settlementHandshake) Do(tm *Manager, tr Transport, timeout time.Duration) (entry *Entry, err error) {
	done := make(chan struct{})
	go func() {
		entry, err = handshake(tm, tr)
		close(done)
	}()
	select {
	case <-done:
		return entry, err
	case <-time.After(timeout):
		tm.Logger.Infof("handshake.Do timeout exceeded for value: %v", timeout)
		return nil, errors.New("deadline exceeded on handshake")
	}
}

func makeEntry(tp Transport, public bool) *Entry {
	return &Entry{
		ID:        MakeTransportID(tp.LocalPK(), tp.RemotePK(), tp.Type(), public),
		LocalKey:  tp.LocalPK(),
		RemoteKey: tp.RemotePK(),
		Type:      tp.Type(),
		Public:    public,
	}
}

func compareEntries(expected, received *Entry, checkPublic bool) error {
	if !checkPublic {
		expected.Public = received.Public
		expected.ID = MakeTransportID(expected.LocalKey, expected.RemoteKey, expected.Type, expected.Public)
	}
	if expected.ID != received.ID {
		return errors.New("received entry's 'tp_id' is not of expected")
	}
	if expected.LocalKey != received.LocalKey {
		return errors.New("received entry's 'local_pk' is not of expected")
	}
	if expected.RemoteKey != received.RemoteKey {
		return errors.New("received entry's 'remote_pk' is not of expected")
	}
	if expected.Type != received.Type {
		return errors.New("received entry's 'type' is not of expected")
	}
	if expected.Public != received.Public {
		return errors.New("received entry's 'public' is not of expected")
	}
	return nil
}

func receiveAndVerifyEntry(r io.Reader, expected *Entry, remotePK cipher.PubKey, checkPublic bool) (*SignedEntry, error) {
	var recvSE SignedEntry
	if err := json.NewDecoder(r).Decode(&recvSE); err != nil {
		return nil, fmt.Errorf("failed to read entry: %s", err)
	}
	if err := compareEntries(expected, recvSE.Entry, checkPublic); err != nil {
		return nil, err
	}
	sig, ok := recvSE.Signature(remotePK)
	if !ok {
		return nil, errors.New("invalid remote signature")
	}
	if err := cipher.VerifyPubKeySignedPayload(remotePK, sig, recvSE.Entry.ToBinary()); err != nil {
		return nil, err
	}
	return &recvSE, nil
}

func settlementInitiatorHandshake(public bool) settlementHandshake {
	return func(tm *Manager, tp Transport) (*Entry, error) {
		entry := makeEntry(tp, public)
		se, ok := NewSignedEntry(entry, tm.config.PubKey, tm.config.SecKey)
		if !ok {
			return nil, errors.New("failed to sign entry")
		}
		if err := json.NewEncoder(tp).Encode(se); err != nil {
			return nil, fmt.Errorf("failed to write entry: %v", err)
		}
		if _, err := receiveAndVerifyEntry(tp, entry, tp.RemotePK(), true); err != nil {
			return nil, err
		}
		tm.addEntry(entry)
		return entry, nil
	}
}

func settlementResponderHandshake() settlementHandshake {
	return func(tm *Manager, tr Transport) (*Entry, error) {
		expectedEntry := makeEntry(tr, false)
		recvSignedEntry, err := receiveAndVerifyEntry(tr, expectedEntry, tr.RemotePK(), false)
		if err != nil {
			return nil, err
		}
		if ok := recvSignedEntry.Sign(tm.Local(), tm.config.SecKey); !ok {
			return nil, errors.New("failed to sign received entry")
		}
		if isNew := tm.addIfNotExist(expectedEntry); !isNew {
			_, err = tm.config.DiscoveryClient.UpdateStatuses(context.Background(), &Status{ID: recvSignedEntry.Entry.ID, IsUp: true})
		} else {
			err = tm.config.DiscoveryClient.RegisterTransports(context.Background(), recvSignedEntry)
		}
		if err != nil {
			return nil, err
		}
		if err := json.NewEncoder(tr).Encode(recvSignedEntry); err != nil {
			return nil, fmt.Errorf("failed to write entry: %s", err)
		}
		return expectedEntry, nil
	}
}
