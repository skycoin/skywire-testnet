package transport_test

import (
	"context"
	"fmt"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport"
)

func ExampleNewDiscoveryMock() {
	dc := transport.NewDiscoveryMock()
	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()
	entry := &transport.Entry{Type: "mock", LocalKey: pk1, RemoteKey: pk2}

	sEntry := &transport.SignedEntry{Entry: entry}

	if err := dc.RegisterTransports(context.TODO(), sEntry); err == nil {
		fmt.Println("RegisterTransport success")
	} else {
		fmt.Println(err.Error())
	}

	if entryWS, err := dc.GetTransportByID(context.TODO(), sEntry.Entry.ID); err == nil {
		fmt.Println("GetTransportByID success")
		fmt.Printf("entryWS.Entry.ID == sEntry.Entry.ID is %v\n", entryWS.Entry.ID == sEntry.Entry.ID)
	} else {
		fmt.Printf("%v", entryWS)
	}

	if entriesWS, err := dc.GetTransportsByEdge(context.TODO(), entry.LocalPK()); err == nil {
		fmt.Println("GetTransportsByEdge success")
		fmt.Printf("entriesWS[0].Entry.LocalPK() == entry.LocalPK() is %v\n", entriesWS[0].Entry.LocalPK() == entry.LocalPK())
	}

	if _, err := dc.UpdateStatuses(context.TODO(), &transport.Status{}); err == nil {
		fmt.Println("UpdateStatuses success")
	} else {
		fmt.Println(err.Error())
	}

	// Output: RegisterTransport success
	// GetTransportByID success
	// entryWS.Entry.ID == sEntry.Entry.ID is true
	// GetTransportsByEdge success
	// entriesWS[0].Entry.LocalPK() == entry.LocalPK() is true
	// UpdateStatuses success
}
