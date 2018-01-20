package monitor

import (
	"io/ioutil"
	"encoding/json"
	"path/filepath"
	"os"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Pass string
}

func readUserConfig(path string) (user *User, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	user = &User{}
	err = json.Unmarshal(fb, user)
	return
}

func WriteConfig(data []byte, path string) (err error) {
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, data, 0600)
	return
}

func checkPass(pass string) (err error) {
	user, err := readUserConfig(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			user = &User{Pass: getBcrypt("1234")}
			data := []byte("")
			data, err = json.Marshal(user)
			if err != nil {
				return
			}
			err = WriteConfig(data, userPath)
			if err != nil {
				return
			}
		} else {
			return
		}
	}
	if !matchPassword(user.Pass, pass) {
		err = errors.New("authentication failed")
		return
	}
	return
}

//bcrypt pass
func getBcrypt(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

//match pass
func matchPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true
	}
	return false
}
