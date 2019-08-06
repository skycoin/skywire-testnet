package transport_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/transport"
)

func TestNewEntry(t *testing.T) {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB := transport.NewEntry(pkA, pkB, "", true)
	entryBA := transport.NewEntry(pkA, pkB, "", true)

	assert.True(t, entryAB.Edges == entryBA.Edges)
	assert.True(t, entryAB.ID == entryBA.ID)
	assert.NotNil(t, entryAB.ID)
	assert.NotNil(t, entryBA.ID)
}

func TestEntry_SetEdges(t *testing.T) {
	pkA, _ := cipher.GenerateKeyPair()
	pkB, _ := cipher.GenerateKeyPair()

	entryAB, entryBA := transport.Entry{}, transport.Entry{}

	entryAB.SetEdges(pkA, pkB)
	entryBA.SetEdges(pkA, pkB)

	assert.True(t, entryAB.Edges == entryBA.Edges)
	assert.True(t, entryAB.ID == entryBA.ID)
	assert.NotNil(t, entryAB.ID)
	assert.NotNil(t, entryBA.ID)
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
