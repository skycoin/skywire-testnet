package therealssh

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/skycoin/skycoin/src/util/logging"
)

var log = logging.MustGetLogger("therealssh")

// Session represents PTY sessions. Channel normally handles Session's lifecycle.
type Session struct {
	pty, tty *os.File

	user *user.User
	cmd  *exec.Cmd
}

// OpenSession constructs new PTY sessions.
func OpenSession(user *user.User, sz *pty.Winsize) (s *Session, err error) {
	s = &Session{user: user}
	s.pty, s.tty, err = pty.Open()
	if err != nil {
		err = fmt.Errorf("failed to open PTY: %s", err)
		return
	}

	if sz == nil {
		return
	}

	if err = pty.Setsize(s.pty, sz); err != nil {
		if closeErr := s.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("Failed to close session")
		}
		err = fmt.Errorf("failed to set PTY size: %s", err)
	}

	return
}

// Start executes command on Session's PTY.
func (s *Session) Start(command string) (err error) {
	defer func() {
		if err := s.tty.Close(); err != nil {
			log.WithError(err).Warn("Failed to close TTY")
		}
	}()

	if command == "shell" {
		if command, err = resolveShell(s.user); err != nil {
			return err
		}
	}

	components := strings.Split(command, " ")
	cmd := exec.Command(components[0], components[1:]...) // nolint:gosec
	cmd.Dir = s.user.HomeDir
	cmd.Stdout = s.tty
	cmd.Stdin = s.tty
	cmd.Stderr = s.tty
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Credential = s.credentials()

	s.cmd = cmd
	return cmd.Start()
}

// Run executes a command and returns it's output and error if any
func (s *Session) Run(command string) ([]byte, error) {
	var err error

	if command == "shell" {
		if command, err = resolveShell(s.user); err != nil {
			return nil, err
		}
	}

	components := strings.Split(command, " ")

	c := exec.Command(components[0],components[1:]...)
	ptmx, err := pty.Start(c)
	if err != nil {
		return nil, err
	}

	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// as stated in https://github.com/creack/pty/issues/21#issuecomment-513069505 we can ignore this error
	res, _ := ioutil.ReadAll(ptmx) // nolint: err
	return res, nil
}

// Wait for pty process to exit.
func (s *Session) Wait() error {
	if s.cmd == nil {
		return nil
	}

	return s.cmd.Wait()
}

// WindowChange resize PTY Session size.
func (s *Session) WindowChange(sz *pty.Winsize) error {
	if err := pty.Setsize(s.pty, sz); err != nil {
		return fmt.Errorf("failed to set PTY size: %s", err)
	}

	return nil
}

func (s *Session) credentials() *syscall.Credential {
	if s.user == nil {
		return nil
	}

	u, err := user.Current()
	if err != nil {
		return nil
	}

	if u.Uid == s.user.Uid {
		return nil
	}

	uid, err := strconv.Atoi(s.user.Uid)
	if err != nil {
		return nil
	}

	gid, err := strconv.Atoi(s.user.Gid)
	if err != nil {
		return nil
	}

	return &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
}

func (s *Session) Write(p []byte) (int, error) {
	return s.pty.Write(p)
}

func (s *Session) Read(p []byte) (int, error) {
	return s.pty.Read(p)
}

// Close releases PTY resources.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	return s.pty.Close()
}
