package commands

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/skycoin/dmsg/cipher"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	ssh "github.com/skycoin/skywire/internal/therealssh"
)

var rpcAddr string

var rootCmd = &cobra.Command{
	Use:   "SSH-cli [user@]remotePK [command] [args...]",
	Short: "Client for the SSH-client app",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		client, err := rpc.DialHTTP("tcp", rpcAddr)
		if err != nil {
			log.Fatal("RPC connection failed:", err)
		}

		size, err := pty.GetsizeFull(os.Stdin)
		if err != nil {
			log.Fatal("Failed to get TTY size:", err)
		}

		username, pk, err := resolveUser(args[0])
		if err != nil {
			log.Fatal("Invalid user/pk pair: ", err)
		}

		remotePK := cipher.PubKey{}
		if err := remotePK.UnmarshalText([]byte(pk)); err != nil {
			log.Fatal("Invalid remote PubKey: ", err)
		}

		ptyArgs := &ssh.RequestPTYArgs{Username: username, RemotePK: remotePK, Size: size}
		var channelID uint32
		if err := client.Call("RPCClient.RequestPTY", ptyArgs, &channelID); err != nil {
			log.Fatal("Failed to request PTY:", err)
		}
		defer func() {
			if err := client.Call("RPCClient.Close", &channelID, nil); err != nil {
				log.Printf("Failed to close RPC client: %v", err)
			}
		}()

		var socketPath string
		execArgs := &ssh.ExecArgs{ChannelID: channelID, CommandWithArgs: args[1:]}
		if err := client.Call("RPCClient.Exec", execArgs, &socketPath); err != nil {
			log.Fatal("Failed to start shell:", err)
		}

		var conn net.Conn
		for i := 0; i < 5; i++ {
			conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
			if err == nil {
				break
			}

			time.Sleep(time.Second)
		}
		if err != nil {
			log.Fatal("Failed dial ssh socket:", err)
		}

		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				size, err := pty.GetsizeFull(os.Stdin)
				if err != nil {
					log.Println("Failed to change pty size:", err)
					return
				}

				var result int
				if err := client.Call("RPCClient.WindowChange", &ssh.WindowChangeArgs{ChannelID: channelID, Size: size}, &result); err != nil {
					log.Println("Failed to change pty size:", err)
				}
			}
		}()

		oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatal("Failed to set terminal to raw mode:", err)
		}
		defer func() {
			if err := terminal.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				log.Printf("Failed to restore terminal: %v", err)
			}
		}()

		go func() {
			if _, err := io.Copy(conn, os.Stdin); err != nil {
				log.Fatal("Failed to write to ssh socket:", err)
			}
		}()

		if _, err := io.Copy(os.Stdout, conn); err != nil {
			log.Fatal("Failed to read from ssh socket:", err)
		}
	},
}

func resolveUser(arg string) (username string, pk string, err error) {
	components := strings.Split(arg, "@")
	if len(components) == 1 {
		if u, uErr := user.Current(); uErr != nil {
			err = fmt.Errorf("failed to resolve current user: %s", uErr)
		} else {
			username = u.Username
			pk = components[0]
		}

		return
	}

	username = components[0]
	pk = components[1]
	return
}

func init() {
	rootCmd.Flags().StringVarP(&rpcAddr, "rpc", "", ":2222", "RPC address to connect to")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runInPTY() error {
	c := exec.Command("sh")
	ptmx, err := pty.Start(c)

	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.

	// Set stdin in raw mode.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}