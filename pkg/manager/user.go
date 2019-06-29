package manager

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"go.etcd.io/bbolt"
)

const (
	boltTimeout        = 10 * time.Second
	boltUserBucketName = "users"
	passwordSaltLen    = 16
)

func init() {
	gob.Register(User{})
}

// User represents a user of the manager.
type User struct {
	Name   string
	PwSalt []byte
	PwHash cipher.SHA256
}

// SetName checks the provided name, and sets the name if format is valid.
func (u *User) SetName(name string) bool {
	if !UsernameFormatOkay(name) {
		return false
	}
	u.Name = name
	return true
}

// SetPassword checks the provided password, and sets the password if format is valid.
func (u *User) SetPassword(password string) bool {
	if !PasswordFormatOkay(password) {
		return false
	}
	u.PwSalt = cipher.RandByte(passwordSaltLen)
	u.PwHash = cipher.SumSHA256(append([]byte(password), u.PwSalt...))
	return true
}

// VerifyPassword verifies the password input with hash and salt.
func (u *User) VerifyPassword(password string) bool {
	return cipher.SumSHA256(append([]byte(password), u.PwSalt...)) == u.PwHash
}

// Encode encodes the user to bytes.
func (u *User) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(u); err != nil {
		catch(err, "unexpected user encode error:")
	}
	return buf.Bytes()
}

// DecodeUser decodes the user from bytes.
func DecodeUser(raw []byte) User {
	var user User
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&user); err != nil {
		catch(err, "unexpected decode user error:")
	}
	return user
}

// UserStore stores users.
type UserStore interface {
	User(name string) (User, bool)
	AddUser(user User) bool
	SetUser(user User) bool
	RemoveUser(name string)
}

// BoltUserStore implements UserStore, storing users in a bbolt database file.
type BoltUserStore struct {
	*bbolt.DB
}

// NewBoltUserStore creates a new BoltUserStore.
func NewBoltUserStore(path string) (*BoltUserStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0700)); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(path, os.FileMode(0600), &bbolt.Options{Timeout: boltTimeout})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(boltUserBucketName))
		return err
	})
	return &BoltUserStore{DB: db}, err
}

// User obtains a single user. Returns true if user exists.
func (s *BoltUserStore) User(name string) (user User, ok bool) {
	catch(s.View(func(tx *bbolt.Tx) error { //nolint:unparam
		users := tx.Bucket([]byte(boltUserBucketName))
		rawUser := users.Get([]byte(name))
		if rawUser == nil {
			ok = false
			return nil
		}
		user = DecodeUser(rawUser)
		ok = true
		return nil
	}))
	return user, ok
}

// AddUser adds a new user; ok is true when successful.
func (s *BoltUserStore) AddUser(user User) (ok bool) {
	catch(s.Update(func(tx *bbolt.Tx) error {
		users := tx.Bucket([]byte(boltUserBucketName))
		if users.Get([]byte(user.Name)) != nil {
			ok = false
			return nil
		}
		ok = true
		return users.Put([]byte(user.Name), user.Encode())
	}))
	return ok
}

// SetUser changes an existing user. Returns true on success.
func (s *BoltUserStore) SetUser(user User) (ok bool) {
	catch(s.Update(func(tx *bbolt.Tx) error {
		users := tx.Bucket([]byte(boltUserBucketName))
		if users.Get([]byte(user.Name)) == nil {
			ok = false
			return nil
		}
		ok = true
		return users.Put([]byte(user.Name), user.Encode())
	}))
	return ok
}

// RemoveUser removes a user of given username.
func (s *BoltUserStore) RemoveUser(name string) {
	catch(s.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(boltUserBucketName)).Delete([]byte(name))
	}))
}

// SingleUserStore implements UserStore while enforcing only having a single user.
type SingleUserStore struct {
	username string
	UserStore
}

// NewSingleUserStore creates a new SingleUserStore with provided username and UserStore.
func NewSingleUserStore(username string, users UserStore) *SingleUserStore {
	return &SingleUserStore{
		username:  username,
		UserStore: users,
	}
}

// User gets a user.
func (s *SingleUserStore) User(name string) (User, bool) {
	if s.allowName(name) {
		return s.UserStore.User(name)
	}
	return User{}, false
}

// AddUser adds a new user.
func (s *SingleUserStore) AddUser(user User) bool {
	if s.allowName(user.Name) {
		return s.UserStore.AddUser(user)
	}
	return false
}

// SetUser sets an existing user.
func (s *SingleUserStore) SetUser(user User) bool {
	if s.allowName(user.Name) {
		return s.UserStore.SetUser(user)
	}
	return false
}

// RemoveUser removes a user.
func (s *SingleUserStore) RemoveUser(name string) {
	if s.allowName(name) {
		s.UserStore.RemoveUser(name)
	}
}

func (s *SingleUserStore) allowName(name string) bool {
	return name == s.username
}

// UsernameFormatOkay checks if the username format is valid.
func UsernameFormatOkay(name string) bool {
	return regexp.MustCompile(`^[a-z0-9_-]{4,21}$`).MatchString(name)
}

// PasswordFormatOkay checks if the password format is valid.
func PasswordFormatOkay(pass string) bool {
	if len(pass) < 6 || len(pass) > 64 {
		return false
	}
	// TODO: implement more advanced password checking.
	return true
}
