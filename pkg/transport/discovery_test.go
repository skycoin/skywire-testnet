package transport_test

import (
	"context"
	"fmt"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport"
)

func ExampleNewDiscoveryMock() {
	dc := transport.NewDiscoveryMock()
	pk1, _ := cipher.GenerateKeyPair()
	pk2, _ := cipher.GenerateKeyPair()
	entry := &transport.Entry{Type: "mock", Purpose: dmsg.PurposeTest, EdgeKeys: transport.SortPubKeys(pk1, pk2)}

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

	if entriesWS, err := dc.GetTransportsByEdge(context.TODO(), entry.Edges()[0]); err == nil {
		fmt.Println("GetTransportsByEdge success")
		fmt.Printf("entriesWS[0].Entry.Edges()[0] == entry.Edges()[0] is %v\n", entriesWS[0].Entry.Edges()[0] == entry.Edges()[0])
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
	// entriesWS[0].Entry.Edges()[0] == entry.Edges()[0] is true
	// UpdateStatuses success
}
