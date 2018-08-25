package monitor

import (
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
	"encoding/hex"
	"math/rand"
)

type User struct {
	Pass string
}

var user *User

func readUserConfig(path string) (user *User, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	user = &User{}
	err = json.Unmarshal(fb, user)
	return
}

func WriteConfig(user *User, path string) (err error) {
	data, err := json.Marshal(user)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, data, 0600)
	return
}

func checkPass(pass string) (err error) {
	user, err = readUserConfig(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			user = &User{Pass: getBcrypt("1234")}
			err = WriteConfig(user, userPath)
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

func isDefaultPass() bool {
	if user == nil {
		user, _ = readUserConfig(userPath)
	}
	return matchPassword(user.Pass, "1234")
}

func getRandomString(len int) string {
	bytes := make([]byte, len)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Read(bytes)
	return hex.EncodeToString(bytes)
}