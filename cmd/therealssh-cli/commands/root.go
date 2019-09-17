package commands

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	ssh "github.com/SkycoinProject/skywire/pkg/therealssh"
)

var (
	rpcAddr       string
	ptyMode       bool
	ptyRows       uint16
	ptyCols       uint16
	ptyX          uint16
	ptyY          uint16
	ptyBufferSize uint32
)

var rootCmd = &cobra.Command{
	Use:   "SSH-cli [user@]remotePK [command] [args...]",
	Short: "Client for the SSH-client app",
	Args:  cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		if ptyMode {
			runInPTY(args)
			return
		}
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

		conn, err := dialUnix(socketPath)
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

func dialUnix(socketPath string) (net.Conn, error) {
	var conn net.Conn
	var err error

	for i := 0; i < 5; i++ {
		conn, err = net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
		if err == nil {
			break
		}

		time.Sleep(time.Second)
	}

	return conn, err
}

func runInPTY(args []string) {
	client, err := rpc.DialHTTP("tcp", rpcAddr)
	if err != nil {
		log.Fatal("RPC connection failed:", err)
	}

	username, pk, err := resolveUser(args[0])
	if err != nil {
		log.Fatal("Invalid user/pk pair: ", err)
	}

	remotePK := cipher.PubKey{}
	if err := remotePK.UnmarshalText([]byte(pk)); err != nil {
		log.Fatal("Invalid remote PubKey: ", err)
	}

	ptyArgs := &ssh.RequestPTYArgs{
		Username: username,
		RemotePK: remotePK,
		Size: &pty.Winsize{
			Rows: uint16(ptyRows),
			Cols: ptyCols,
			X:    ptyX,
			Y:    ptyY,
		},
	}

	var channelID uint32
	if err := client.Call("RPCClient.RequestPTY", ptyArgs, &channelID); err != nil {
		log.Fatal("Failed to request PTY:", err)
	}

	var socketPath string
	execArgs := &ssh.ExecArgs{
		ChannelID:       channelID,
		CommandWithArgs: args[1:],
	}

	err = client.Call("RPCClient.Run", execArgs, &socketPath)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := dialUnix(socketPath)
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, ptyBufferSize)
	_, err = conn.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(string(b))
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
	rootCmd.Flags().BoolVarP(&ptyMode, "pty", "", false, "Whether to run the command in a simulated PTY or not")
	rootCmd.Flags().Uint16VarP(&ptyRows, "ptyrows", "", 100, "PTY Rows. Applicable if run with pty flag")
	rootCmd.Flags().Uint16VarP(&ptyCols, "ptycols", "", 100, "PTY Cols. Applicable if run with pty flag")
	rootCmd.Flags().Uint16VarP(&ptyX, "ptyx", "", 100, "PTY X. Applicable if run with pty flag")
	rootCmd.Flags().Uint16VarP(&ptyY, "ptyy", "", 100, "PTY Y. Applicable if run with pty flag")
	rootCmd.Flags().Uint32VarP(&ptyBufferSize, "ptybuffer", "", 1024, "PTY Buffer size to store command output")
}

// Execute executes root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
