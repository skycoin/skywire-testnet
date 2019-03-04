// +build integration

package client_test

import (
	"context"
	"fmt"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
	"github.com/skycoin/skywire/pkg/transport"
)

func Example() {
	// Connects the client to a local messaging-discovery server listening in port 8080
	apiClient := client.New("http://localhost:8080")

	// Create keypairs to use with the Client entry
	pk, sk := transport.GenerateDeterministicKeyPair([]byte(`example`))

	// Create client metadata
	clientData := client.NewClient(nil, nil)

	// Create ephemeral keys metadata
	ephemeralKeys := []client.ServerKeys{
		client.NewEphemeralKeys("ephemeralKey1", "staticServerKey"),
		client.NewEphemeralKeys("ephemeralKey2", "staticServerKey"),
	}

	// Create client entry, which iteration sequence is 0
	entry := client.NewClientEntry(pk, 0, clientData, ephemeralKeys)

	// Use the secret key to sign the entry
	entry.Sign(sk)

	// Use the client to set the new entry in the messaging-discovery server
	err := apiClient.SetEntry(context.TODO(), entry)
	if err != nil {
		panic(err)
	}

	// Use the client to retrieve the entry associated to the public key
	retrievedEntry, err := apiClient.Entry(context.TODO(), pk)
	if err != nil {
		panic(err)
	}

	fmt.Println("original entry pk: ", entry.Keys.Static)
	fmt.Println("recovered entry pk: ", retrievedEntry.Keys.Static)

	// You can also update the entry calling update entry method
	// Internally it will update the sequence and re-sign the entry before calling set
	entry.Version = "1"
	apiClient.UpdateEntry(context.TODO(), sk, entry)

	retrievedEntry, err = apiClient.Entry(context.TODO(), pk)
	if err != nil {
		panic(err)
	}

	fmt.Println("version of retrieved entry: ", retrievedEntry.Version)

	// Create a server entry, this one will update the previous entry
	// associated with the public key, we set the iteration sequence to 1
	serverData := client.NewServer("localhost:8080", 5)
	serverEntry := client.NewServerEntry(pk, 1, serverData)
	serverEntry.Sign(sk)

	// Update the server entry
	err = apiClient.UpdateEntry(context.TODO(), sk, serverEntry)
	if err != nil {
		panic(err)
	}

	// Use the client library to retrieve an array of server entries from the messaging-discovery server
	entries, err := apiClient.AvailableServers(context.TODO())
	if err != nil {
		panic(err)
	}

	fmt.Println("server entry pk: ", serverEntry.Keys.Static)
	fmt.Println("retrieved server entry pk: ", entries[0].Keys.Static)

	// Output:
	// original entry pk:  02ca451b007ee2f00324fb95475d5a194b1b7a15dbf61c728ec97168ad03f9bdd8
	// recovered entry pk:  02ca451b007ee2f00324fb95475d5a194b1b7a15dbf61c728ec97168ad03f9bdd8
	// version of retrieved entry:  1
	// server entry pk:  02ca451b007ee2f00324fb95475d5a194b1b7a15dbf61c728ec97168ad03f9bdd8
	// retrieved server entry pk:  02ca451b007ee2f00324fb95475d5a194b1b7a15dbf61c728ec97168ad03f9bdd8
}
