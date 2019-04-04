package transport

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/skycoin/skywire/pkg/cipher"
)

// ExampleNewEntry shows that with different order of edges:
// - Entry.ID is the same
// - Edges() call is the same
func ExampleNewEntry() {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB := NewEntry(pkA, pkB, "", true)
	entryBA := NewEntry(pkB, pkA, "", true)

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

	entryAB := Entry{
		ID:       uuid.UUID{},
		EdgeKeys: [2]cipher.PubKey{pkA, pkB},
		Type:     "",
		Public:   true,
	}

	entryBA := Entry{
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

	entryAB, entryBA := Entry{}, Entry{}

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

	entry := NewEntry(pkA, pkB, "mock", true)
	sEntry := &SignedEntry{Entry: entry}

	if sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null() {
		fmt.Println("No signatures set")
	}

	sEntry.Sign(pkA, skA)
	if (!sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null()) ||
		(!sEntry.Signatures[1].Null() && sEntry.Signatures[0].Null()) {
		fmt.Println("One signature set")
	}

	sEntry.Sign(pkB, skB)
	if !sEntry.Signatures[0].Null() && !sEntry.Signatures[1].Null() {
		fmt.Println("Both signatures set")
	}

	// Output: No signatures set
	// One signature set
	// Both signatures set
}

func ExampleSignedEntry_Signature() {
	pkA, skA := cipher.GenerateKeyPair()
	pkB, skB := cipher.GenerateKeyPair()

	entry := NewEntry(pkA, pkB, "mock", true)
	sEntry := &SignedEntry{Entry: entry}
	sEntry.Sign(pkA, skA)
	sEntry.Sign(pkB, skB)

	if sEntry.Signature(pkA) == sEntry.Signatures[sEntry.Index(pkA)] {
		fmt.Println("SignatureA got")
	}
	if sEntry.Signature(pkB) == sEntry.Signatures[sEntry.Index(pkB)] {
		fmt.Println("SignatureB got")
	}

	// Output: SignatureA got
	// SignatureB got
}
