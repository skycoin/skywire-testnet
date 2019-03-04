package therealssh

import (
	"fmt"
	"os/exec"
	"os/user"
	"regexp"
)

func resolveShell(u *user.User) (string, error) {
	dir := "Local/Default/Users/" + u.Username
	out, err := exec.Command("dscl", "localhost", "-read", dir, "UserShell").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("UserShell: (/[^ ]+)\n")
	matched := re.FindStringSubmatch(string(out))
	shell := matched[1]
	if shell == "" {
		return "", fmt.Errorf("Invalid output: %s", string(out))
	}

	return shell, nil
}
