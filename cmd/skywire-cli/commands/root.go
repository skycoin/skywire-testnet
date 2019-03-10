package commands

import (
	"fmt"
	"log"
	"net/rpc"
	"strconv"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/manager"
	"github.com/skycoin/skywire/pkg/node"
)

var rpcAddr string

var rootCmd = &cobra.Command{
	Use:   "skywire-cli",
	Short: "Command Line Interface for skywire",
}

// Execute executes root CLI command.
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&rpcAddr, "rpc", "", "localhost:3435", "RPC server address")
	rootCmd.Execute() //nolint:errcheck
}

func rpcClient() node.RPCClient {
	client, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}
	return manager.NewRPCClient(client, node.RPCPrefix)
}

func catch(err error, msgs ...string) {
	if err != nil {
		if len(msgs) > 0 {
			log.Fatalln(append(msgs, err.Error()))
		} else {
			log.Fatalln(err)
		}
	}
}

type transportID uuid.UUID

// String implements pflag.Value
func (t transportID) String() string { return uuid.UUID(t).String() }

// Type implements pflag.Value
func (transportID) Type() string { return "transportID" }

// Set implements pflag.Value
func (t *transportID) Set(s string) error {
	tID, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*t = transportID(tID)
	return nil
}

func parsePK(name, v string) cipher.PubKey {
	var pk cipher.PubKey
	catch(pk.Set(v), fmt.Sprintf("failed to parse <%s>:", name))
	return pk
}

func parseUUID(name, v string) uuid.UUID {
	id, err := uuid.Parse(v)
	catch(err, fmt.Sprintf("failed to parse <%s>:", name))
	return id
}

func parseUint(name, v string, bitSize int) uint64 {
	i, err := strconv.ParseUint(v, 10, bitSize)
	catch(err, fmt.Sprintf("failed to parse <%s>:", name))
	return i
}
