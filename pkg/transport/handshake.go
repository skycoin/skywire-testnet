package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/SkycoinProject/dmsg/cipher"
)

func makeEntry(pk1, pk2 cipher.PubKey, tpType string) Entry {
	return Entry{
		ID:     MakeTransportID(pk1, pk2, tpType),
		Edges:  SortEdges(pk1, pk2),
		Type:   tpType,
		Public: true,
	}
}

func makeEntryFromTp(tp Transport) Entry {
	return makeEntry(tp.LocalPK(), tp.RemotePK(), tp.Type())
}

func compareEntries(expected, received *Entry) error {
	if expected.ID != received.ID {
		return errors.New("received entry's 'tp_id' is not of expected")
	}
	if expected.Edges != received.Edges {
		return errors.New("received entry's 'edges' is not of expected")
	}
	if expected.Type != received.Type {
		return errors.New("received entry's 'type' is not of expected")
	}
	if expected.Public != received.Public {
		return errors.New("received entry's 'public' is not of expected")
	}
	return nil
}

func receiveAndVerifyEntry(r io.Reader, expected *Entry, remotePK cipher.PubKey) (*SignedEntry, error) {
	var recvSE SignedEntry
	if err := json.NewDecoder(r).Decode(&recvSE); err != nil {
		return nil, fmt.Errorf("failed to read entry: %s", err)
	}
	if err := compareEntries(expected, recvSE.Entry); err != nil {
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

// SettlementHS represents a settlement handshake.
// This is the handshake responsible for registering a transport to transport discovery.
type SettlementHS func(ctx context.Context, dc DiscoveryClient, tp Transport, sk cipher.SecKey) error

// Do performs the settlement handshake.
func (hs SettlementHS) Do(ctx context.Context, dc DiscoveryClient, tp Transport, sk cipher.SecKey) (err error) {
	done := make(chan struct{})
	go func() {
		err = hs(ctx, dc, tp, sk)
		close(done)
	}()
	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MakeSettlementHS creates a settlement handshake.
// `init` determines whether the local side is initiating or responding.
func MakeSettlementHS(init bool) SettlementHS {

	// initiating logic.
	initHS := func(ctx context.Context, dc DiscoveryClient, tp Transport, sk cipher.SecKey) (err error) {
		entry := makeEntryFromTp(tp)

		defer func() { _, _ = dc.UpdateStatuses(ctx, &Status{ID: entry.ID, IsUp: err == nil}) }() //nolint:errcheck

		// create signed entry and send it to responding visor node.
		se, ok := NewSignedEntry(&entry, tp.LocalPK(), sk)
		if !ok {
			return errors.New("failed to sign entry")
		}
		if err := json.NewEncoder(tp).Encode(se); err != nil {
			return fmt.Errorf("failed to write entry: %v", err)
		}

		// await okay signal.
		accepted := make([]byte, 1)
		if _, err := io.ReadFull(tp, accepted); err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}
		if accepted[0] == 0 {
			return fmt.Errorf("transport settlement rejected by remote")
		}
		return nil
	}

	// responding logic.
	respHS := func(ctx context.Context, dc DiscoveryClient, tp Transport, sk cipher.SecKey) error {
		entry := makeEntryFromTp(tp)

		// receive, verify and sign entry.
		recvSE, err := receiveAndVerifyEntry(tp, &entry, tp.RemotePK())
		if err != nil {
			return err
		}
		if ok := recvSE.Sign(tp.LocalPK(), sk); !ok {
			return errors.New("failed to sign received entry")
		}
		entry = *recvSE.Entry

		// Ensure transport is registered.
		_ = dc.RegisterTransports(ctx, recvSE) //nolint:errcheck

		// inform initiating visor node.
		if _, err := tp.Write([]byte{1}); err != nil {
			return fmt.Errorf("failed to accept transport settlement: write failed: %v", err)
		}
		return nil
	}

	if init {
		return initHS
	}
	return respHS
}
