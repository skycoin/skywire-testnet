package manager

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"go.etcd.io/bbolt"

	"github.com/skycoin/skywire/pkg/cipher"
)

const (
	boltTimeout        = 10 * time.Second
	boltUserBucketName = "users"
)

func init() {
	gob.Register(User{})
}

type User struct {
	Name   string
	PwSalt []byte
	PwHash cipher.SHA256
}

func (u *User) SetName(pattern, name string) bool {
	if pattern != "" {
		ok, err := regexp.MatchString(pattern, name)
		catch(err, "invalid username regex:")
		if !ok {
			return false
		}
	}
	u.Name = name
	return true
}

func (u *User) SetPassword(saltLen int, pattern, password string) bool {
	if pattern != "" {
		ok, err := regexp.MatchString(pattern, password)
		if err != nil {
			catch(err, "invalid password regex:")
		}
		if !ok {
			return false
		}
	}
	u.PwSalt = cipher.RandByte(saltLen)
	u.PwHash = cipher.SumSHA256(append([]byte(password), u.PwSalt...))
	return true
}

func (u *User) VerifyPassword(password string) bool {
	return cipher.SumSHA256(append([]byte(password), u.PwSalt...)) == u.PwHash
}

func (u *User) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(u); err != nil {
		catch(err, "unexpected user encode error:")
	}
	return buf.Bytes()
}

func DecodeUser(raw []byte) User {
	var user User
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&user); err != nil {
		catch(err, "unexpected decode user error:")
	}
	return user
}

type UserStore interface {
	User(name string) (User, bool)
	AddUser(user User) bool
	SetUser(user User) bool
	RemoveUser(name string)
}

type BoltUserStore struct {
	*bbolt.DB
}

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

func (s *BoltUserStore) User(name string) (user User, ok bool) {
	catch(s.View(func(tx *bbolt.Tx) error {
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

func (s *BoltUserStore) RemoveUser(name string) {
	catch(s.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(boltUserBucketName)).Delete([]byte(name))
	}))
}

type SingleUserStore struct {
	username string
	UserStore
}

func NewSingleUserStore(username string, users UserStore) *SingleUserStore {
	return &SingleUserStore{
		username:  username,
		UserStore: users,
	}
}

func (s *SingleUserStore) User(name string) (User, bool) {
	if s.allowName(name) {
		return s.UserStore.User(name)
	}
	return User{}, false
}

func (s *SingleUserStore) AddUser(user User) bool {
	if s.allowName(user.Name) {
		return s.UserStore.AddUser(user)
	}
	return false
}

func (s *SingleUserStore) SetUser(user User) bool {
	if s.allowName(user.Name) {
		return s.UserStore.SetUser(user)
	}
	return false
}

func (s *SingleUserStore) RemoveUser(name string) {
	if s.allowName(name) {
		s.UserStore.RemoveUser(name)
	}
}

func (s *SingleUserStore) allowName(name string) bool {
	return name == s.username
}
