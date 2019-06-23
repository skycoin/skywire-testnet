package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/skycoin/skywire/pkg/cipher"
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

func settlementInitiatorHandshake(public bool) settlementHandshake {
	return func(tm *Manager, tr Transport) (*Entry, error) {
		tm.Logger.Info("Entering settlementInitiatorHandshake")
		entry := &Entry{
			ID:       MakeTransportID(tr.Edges()[0], tr.Edges()[1], tr.Type(), public),
			EdgeKeys: tr.Edges(),
			Type:     tr.Type(),
			Public:   public,
		}

		remote, ok := tm.Remote(tr.Edges())
		if !ok {
			return nil, errors.New("configured PubKey not found in edges")
		}

		tm.Logger.Infof("settlementInitiatorHandshake: NewSignedEntry with %v", remote)
		sEntry, ok := NewSignedEntry(entry, tm.config.PubKey, tm.config.SecKey)
		if !ok {
			return nil, errors.New("error creating signed entry")
		}
		tm.Logger.Info("settlementInitiatorHandshake: validateSignedEntry")
		if err := validateSignedEntry(sEntry, tr, tm.config.PubKey); err != nil {
			return nil, fmt.Errorf("settlementInitiatorHandshake NewSignedEntry: %s\n sEntry: %v", err, sEntry)
		}
		tm.Logger.Info("settlementInitiatorHandshake: json.NewEncoder.Encode")
		if err := json.NewEncoder(tr).Encode(sEntry); err != nil {
			return nil, fmt.Errorf("write: %s", err)
		}

		tm.Logger.Info("settlementInitiatorHandshake: json.NewDecoder")
		respSEntry := &SignedEntry{}
		if err := json.NewDecoder(tr).Decode(respSEntry); err != nil {
			return nil, fmt.Errorf("read: %s", err)
		}

		//  Verifying remote signature

		tm.Logger.Info("settlementInitiatorHandshake: verifySig")
		if err := verifySig(respSEntry, remote); err != nil {
			return nil, err
		}
		tm.Logger.Info("settlementInitiatorHandshake: tm.walkEntries")
		newEntry := tm.walkEntries(func(e *Entry) bool { return *e == *respSEntry.Entry }) == nil
		if newEntry {
			tm.addEntry(entry)
		}
		tm.Logger.Info("Exiting settlementInitiatorHandshake")
		return respSEntry.Entry, nil
	}
}

func settlementResponderHandshake(tm *Manager, tr Transport) (*Entry, error) {
	tm.Logger.Info("Entering settlementResponderHandshake")
	sEntry := &SignedEntry{}
	if err := json.NewDecoder(tr).Decode(sEntry); err != nil {
		return nil, fmt.Errorf("read: %s", err)
	}

	remote, ok := tm.Remote(tr.Edges())
	if !ok {
		return nil, errors.New("configured PubKey not found in edges")
	}

	if err := validateSignedEntry(sEntry, tr, remote); err != nil {
		return nil, err
	}

	if ok := sEntry.Sign(tm.Local(), tm.config.SecKey); !ok {
		return nil, errors.New("invalid pubkey for signing entry")
	}

	newEntry := tm.walkEntries(func(e *Entry) bool { return *e == *sEntry.Entry }) == nil

	tm.Logger.Info("Entering settlementResponderHandshake: DiscoveryClient.UpdateStatuses/RegisterTransports")
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

	tm.Logger.Info("Entering settlementResponderHandshake: json.NewEncoder(tr).Encode")
	if err := json.NewEncoder(tr).Encode(sEntry); err != nil {
		return nil, fmt.Errorf("write: %s", err)
	}

	if newEntry {
		tm.addEntry(sEntry.Entry)
	}
	tm.Logger.Info("Exiting settlementResponderHandshake")
	return sEntry.Entry, nil
}

func validateSignedEntry(sEntry *SignedEntry, tr Transport, pk cipher.PubKey) error {
	entry := sEntry.Entry
	if entry.Type != tr.Type() {
		return errors.New("invalid entry type")
	}

	if entry.Edges() != tr.Edges() {
		return errors.New("invalid entry edges")
	}

	// Weak check here
	if sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null() {
		return errors.New("invalid entry signature")
	}

	return verifySig(sEntry, pk)
}

func verifySig(sEntry *SignedEntry, pk cipher.PubKey) error {
	sig, ok := sEntry.Signature(pk)
	if !ok {
		return errors.New("invalid pubkey for retrieving signature")
	}

	return cipher.VerifyPubKeySignedPayload(pk, sig, sEntry.Entry.ToBinary())
}
