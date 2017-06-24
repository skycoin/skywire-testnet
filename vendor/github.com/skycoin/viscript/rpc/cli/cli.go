package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	cm "github.com/skycoin/viscript/rpc/climanager"
)

var cliManager cm.CliManager

const prompt string = "Enter the command (help(h) for commands list):\n> "
const defaultPort string = "7777"

func main() {
	port := defaultPort
	if len(os.Args) >= 2 {
		port = os.Args[1]
	}
	cliManager.Init(port)
	promptCycle()
}

func promptCycle() {
	for !cliManager.SessionEnd {
		newCommand, args := inputFromCli()
		if newCommand == "" {
			continue
		}
		cliManager.CommandDispatcher(strings.ToLower(newCommand), args)
	}
}

func inputFromCli() (command string, args []string) {
	fmt.Printf(prompt)
	command = ""
	args = []string{}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()

	splitInput := strings.Fields(input)
	if len(splitInput) == 0 {
		return
	}

	command = strings.Trim(splitInput[0], " ")
	if len(splitInput) > 1 {
		args = splitInput[1:]
	}
	return
}
