package transport

import (
	"context"
	"fmt"

	"github.com/skycoin/skywire/pkg/cipher"
)

func ExampleNewDiscoveryMock() {
	dc := NewDiscoveryMock()
	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()
	entry := &Entry{Type: "mock", EdgeKeys: SortPubKeys(pk1, pk2)}

	sEntry := &SignedEntry{Entry: entry}

	if Ok := dc.RegisterTransports(context.TODO(), sEntry); Ok == nil {
		fmt.Println("RegisterTransport success")
	} else {
		fmt.Println(Ok.Error())
	}

	if entryWS, Ok := dc.GetTransportByID(context.TODO(), sEntry.Entry.ID); Ok == nil {
		fmt.Println("GetTransportByID success")
		fmt.Printf("entryWS.Entry.ID == sEntry.Entry.ID is %v\n", entryWS.Entry.ID == sEntry.Entry.ID)
	} else {
		fmt.Printf("%v", entryWS)
	}

	if entriesWS, Ok := dc.GetTransportsByEdge(context.TODO(), entry.Edges()[0]); Ok == nil {
		fmt.Println("GetTransportsByEdge success")
		fmt.Printf("entriesWS[0].Entry.Edges()[0] == entry.Edges()[0] is %v\n", entriesWS[0].Entry.Edges()[0] == entry.Edges()[0])
	}

	if _, Ok := dc.UpdateStatuses(context.TODO(), &Status{}); Ok == nil {
		fmt.Println("UpdateStatuses success")
	} else {
		fmt.Println(Ok.Error())
	}

	// Output: RegisterTransport success
	// GetTransportByID success
	// entryWS.Entry.ID == sEntry.Entry.ID is true
	// GetTransportsByEdge success
	// entriesWS[0].Entry.Edges()[0] == entry.Edges()[0] is true
	// UpdateStatuses success
}
