package monitor

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User struct contains the user configuration, specifically the users
// hashed password (Pass)
type User struct {
	Pass string
}

var user *User

// readUserCongig will load (read) the User structure from the
// configuration file specified by path. This function will
// read the JSON from the configuration file into a User struct
// and store it in the global variable user.
func readUserConfig(path string) (user *User, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	user = &User{}
	err = json.Unmarshal(fb, user)
	return
}

// WriteConfig stores the JSON respresentation of the User structure
// within a configuration file in the provided path
// This function will set directory permissions to 0700 and
// file permissions to 0600
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

// checkPass checks the provided password plaintext against
// the stored password hash for the user. If the user has not
// been loaded, it will be loaded from configuration. If the
// configuration file does not exist, the default password
// will be assinged to the user and stored in the config
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

// getBcrypt will return a bcrypt generated password hash from the provided
// password plaintext. This function currently uses the bcrypt MinCost
// to generate the hash.
func getBcrypt(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(hash)
}

// matchPassword compares an hashed password (hash) with a possible
// plaintext version (password) and returns the result as a boolean.
func matchPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true
	}
	return false
}

// userHasDefaultPass checks if the currently loaded User structure uses
// the default password ("1234") or not.
// Note: if the global variable user has not been loaded yet, this function
// will cause it to be loaded from the configuration file.
func userHasDefaultPass() bool {
	if user == nil {
		user, _ = readUserConfig(userPath)
	}
	return matchPassword(user.Pass, "1234")
}

// getRandomString returns a randomly generated string of the requested length (len)
func getRandomString(len int) string {
	bytes := make([]byte, len)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Read(bytes)
	return hex.EncodeToString(bytes)
}

// isDefaultPass checks the provided password cleartext string (pass) against
// the hard coded default password string ("1234") and returns the result.
func isDefaultPass(pass string) bool {
	return pass == "1234"
}
