/*
CLI for skywire node
*/
package main

import (
	"github.com/skycoin/skywire/cmd/skywire-cli/commands"
	_ "github.com/skycoin/skywire/cmd/skywire-cli/commands/node"
)

func main() {
	commands.Execute()
}
