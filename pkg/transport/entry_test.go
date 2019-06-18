package transport_test

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

// ExampleNewEntry shows that with different order of edges:
// - Entry.ID is the same
// - Edges() call is the same
func ExampleNewEntry() {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB := transport.NewEntry(pkA, pkB, "", true)
	entryBA := transport.NewEntry(pkB, pkA, "", true)

	if entryAB.ID == entryBA.ID {
		fmt.Println("entryAB.ID == entryBA.ID")
	}
	if entryAB.Edges() == entryBA.Edges() {
		fmt.Println("entryAB.Edges() == entryBA.Edges()")
	}
	// Output: entryAB.ID == entryBA.ID
	// entryAB.Edges() == entryBA.Edges()
}

func ExampleEntry_Edges() {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB := transport.Entry{
		ID:       uuid.UUID{},
		EdgeKeys: [2]cipher.PubKey{pkA, pkB},
		Type:     "",
		Public:   true,
	}

	entryBA := transport.Entry{
		ID:       uuid.UUID{},
		EdgeKeys: [2]cipher.PubKey{pkB, pkA},
		Type:     "",
		Public:   true,
	}

	if entryAB.EdgeKeys != entryBA.EdgeKeys {
		fmt.Println("entryAB.EdgeKeys != entryBA.EdgeKeys")
	}

	if entryAB.Edges() == entryBA.Edges() {
		fmt.Println("entryAB.Edges() == entryBA.Edges()")
	}

	// Output: entryAB.EdgeKeys != entryBA.EdgeKeys
	// entryAB.Edges() == entryBA.Edges()
}

func ExampleEntry_SetEdges() {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB, entryBA := transport.Entry{}, transport.Entry{}

	entryAB.SetEdges([2]cipher.PubKey{pkA, pkB})
	entryBA.SetEdges([2]cipher.PubKey{pkA, pkB})

	if entryAB.EdgeKeys == entryBA.EdgeKeys {
		fmt.Println("entryAB.EdgeKeys == entryBA.EdgeKeys")
	}

	if (entryAB.ID == entryBA.ID) && (entryAB.ID != uuid.UUID{}) {
		fmt.Println("entryAB.ID != uuid.UUID{}")
		fmt.Println("entryAB.ID == entryBA.ID")
	}

	// Output: entryAB.EdgeKeys == entryBA.EdgeKeys
	// entryAB.ID != uuid.UUID{}
	// entryAB.ID == entryBA.ID
}

func ExampleSignedEntry_Sign() {
	pkA, skA := cipher.GenerateKeyPair()
	pkB, skB := cipher.GenerateKeyPair()

	entry := transport.NewEntry(pkA, pkB, "mock", true)
	sEntry := &transport.SignedEntry{Entry: entry}

	if sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null() {
		fmt.Println("No signatures set")
	}

	if ok := sEntry.Sign(pkA, skA); !ok {
		fmt.Println("error signing with skA")
	}
	if (!sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null()) ||
		(!sEntry.Signatures[1].Null() && sEntry.Signatures[0].Null()) {
		fmt.Println("One signature set")
	}

	if ok := sEntry.Sign(pkB, skB); !ok {
		fmt.Println("error signing with skB")
	}

	if !sEntry.Signatures[0].Null() && !sEntry.Signatures[1].Null() {
		fmt.Println("Both signatures set")
	} else {
		fmt.Printf("sEntry.Signatures:\n%v\n", sEntry.Signatures)
	}

	// Output: No signatures set
	// One signature set
	// Both signatures set
}

func ExampleSignedEntry_Signature() {
	pkA, skA := cipher.GenerateKeyPair()
	pkB, skB := cipher.GenerateKeyPair()

	entry := transport.NewEntry(pkA, pkB, "mock", true)
	sEntry := &transport.SignedEntry{Entry: entry}
	if ok := sEntry.Sign(pkA, skA); !ok {
		fmt.Println("Error signing sEntry with (pkA,skA)")
	}
	if ok := sEntry.Sign(pkB, skB); !ok {
		fmt.Println("Error signing sEntry with (pkB,skB)")
	}

	idxA := sEntry.Index(pkA)
	idxB := sEntry.Index(pkB)

	sigA, okA := sEntry.Signature(pkA)
	sigB, okB := sEntry.Signature(pkB)

	if okA && sigA == sEntry.Signatures[idxA] {
		fmt.Println("SignatureA got")
	}

	if okB && (sigB == sEntry.Signatures[idxB]) {
		fmt.Println("SignatureB got")
	}

	// Incorrect case
	pkC, _ := cipher.GenerateKeyPair()
	if _, ok := sEntry.Signature(pkC); !ok {
		fmt.Printf("SignatureC got error: invalid pubkey")
	}

	//
	// Output: SignatureA got
	// SignatureB got
	// SignatureC got error: invalid pubkey
}
