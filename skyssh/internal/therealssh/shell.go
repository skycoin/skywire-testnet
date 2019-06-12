// +build dragonfly freebsd linux netbsd openbsd

package therealssh

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

func resolveShell(u *user.User) (string, error) {
	out, err := exec.Command("getent", "passwd", u.Uid).Output() // nolint
	if err != nil {
		return "", fmt.Errorf("getent failure: %s", err)
	}

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	return ent[6], nil
}
