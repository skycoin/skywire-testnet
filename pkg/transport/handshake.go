package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
)

type settlementHandshake func(tm *Manager, tr Transport) (*Entry, error)

func (handshake settlementHandshake) Do(tm *Manager, tr Transport, timeout time.Duration) (*Entry, error) {
	var entry *Entry
	errCh := make(chan error, 1)
	go func() {
		e, err := handshake(tm, tr)
		entry = e
		errCh <- err
	}()
	select {
	case err := <-errCh:
		return entry, err
	case <-time.After(timeout):
		return nil, errors.New("deadline exceeded")
	}
}

func settlementInitiatorHandshake(id uuid.UUID, public bool) settlementHandshake {
	return func(tm *Manager, tr Transport) (*Entry, error) {
		entry := &Entry{
			ID:     id,
			edges:  tr.Edges(),
			Type:   tr.Type(),
			Public: public,
		}

		newEntry := id == uuid.UUID{}
		if newEntry {
			entry.ID = GetTransportUUID(entry.Edges()[0], entry.Edges()[1], entry.Type)
		}

		sEntry := &SignedEntry{Entry: entry, Signatures: [2]cipher.Sig{entry.Signature(tm.config.SecKey)}}
		if err := json.NewEncoder(tr).Encode(sEntry); err != nil {
			return nil, fmt.Errorf("write: %s", err)
		}

		if err := json.NewDecoder(tr).Decode(sEntry); err != nil {
			return nil, fmt.Errorf("read: %s", err)
		}

		if remote, Ok := tm.Remote(tr.Edges()); Ok == nil {
			if err := verifySig(sEntry, 1, remote); err != nil {
				return nil, err
			}
		} else {
			return nil, Ok
		}

		if newEntry {
			tm.addEntry(entry)
		}

		return sEntry.Entry, nil
	}
}

func settlementResponderHandshake(tm *Manager, tr Transport) (*Entry, error) {
	sEntry := &SignedEntry{}
	if err := json.NewDecoder(tr).Decode(sEntry); err != nil {
		return nil, fmt.Errorf("read: %s", err)
	}

	if remote, Ok := tm.Remote(tr.Edges()); Ok == nil {
		if err := validateEntry(sEntry, tr, remote); err != nil {
			return nil, err
		}
	} else {
		return nil, Ok
	}

	sEntry.Signatures[1] = sEntry.Entry.Signature(tm.config.SecKey)

	newEntry := tm.walkEntries(func(e *Entry) bool { return *e == *sEntry.Entry }) == nil

	var err error
	if sEntry.Entry.Public {
		if !newEntry {
			_, err = tm.config.DiscoveryClient.UpdateStatuses(context.Background(), &Status{ID: sEntry.Entry.ID, IsUp: true})
		} else {
			err = tm.config.DiscoveryClient.RegisterTransports(context.Background(), sEntry)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("entry set: %s", err)
	}

	if err := json.NewEncoder(tr).Encode(sEntry); err != nil {
		return nil, fmt.Errorf("write: %s", err)
	}

	if newEntry {
		tm.addEntry(sEntry.Entry)
	}

	return sEntry.Entry, nil
}

func validateEntry(sEntry *SignedEntry, tr Transport, rpk cipher.PubKey) error {
	entry := sEntry.Entry
	if entry.Type != tr.Type() {
		return errors.New("invalid entry type")
	}

	if entry.Edges() != tr.Edges() {
		return errors.New("invalid entry edges")
	}

	if sEntry.Signatures[0].Null() {
		return errors.New("invalid entry signature")
	}

	return verifySig(sEntry, 0, rpk)
}

func verifySig(sEntry *SignedEntry, idx int, pk cipher.PubKey) error {
	return cipher.VerifyPubKeySignedPayload(pk, sEntry.Signatures[idx], sEntry.Entry.ToBinary())
}
