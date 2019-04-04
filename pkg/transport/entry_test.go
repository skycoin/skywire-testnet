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

	if errA := sEntry.Sign(pkA, skA); errA != nil {
		fmt.Println(errA.Error())
	}
	if (!sEntry.Signatures[0].Null() && sEntry.Signatures[1].Null()) ||
		(!sEntry.Signatures[1].Null() && sEntry.Signatures[0].Null()) {
		fmt.Println("One signature set")
	}

	if errB := sEntry.Sign(pkB, skB); errB != nil {
		fmt.Println(errB.Error())
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

func errorsPrint(errs ...error) {
	for _, err := range errs {
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func ExampleSignedEntry_Signature() {
	pkA, skA := cipher.GenerateKeyPair()
	pkB, skB := cipher.GenerateKeyPair()

	entry := NewEntry(pkA, pkB, "mock", true)
	sEntry := &SignedEntry{Entry: entry}
	if errA := sEntry.Sign(pkA, skA); errA != nil {
		fmt.Println(errA.Error())
	}
	if errB := sEntry.Sign(pkB, skB); errB != nil {
		fmt.Println(errB.Error())
	}

	idxA, errIdxA := sEntry.Index(pkA)
	idxB, errIdxB := sEntry.Index(pkB)

	sigA, errSigA := sEntry.Signature(pkA)
	sigB, errSigB := sEntry.Signature(pkB)

	if sigA == sEntry.Signatures[idxA] {
		fmt.Println("SignatureA got")
	}

	if sigB == sEntry.Signatures[idxB] {
		fmt.Println("SignatureB got")
	}

	// Incorrect case
	pkC, _ := cipher.GenerateKeyPair()
	_, errSigC := sEntry.Signature(pkC)
	if errSigC != nil {
		fmt.Printf("SignatureC got error: %v\n", errSigC.Error())
	}

	errorsPrint(errIdxA, errIdxB, errSigA, errSigB)
	// Output: SignatureA got
	// SignatureB got
	// SignatureC got error: invalid pubkey
}
