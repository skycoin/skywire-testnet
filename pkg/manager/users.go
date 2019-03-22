package manager

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
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

func (u *User) SetPassword(saltLen int, password string) {
	u.PwSalt = cipher.RandByte(saltLen)
	u.PwHash = cipher.SumSHA256(append([]byte(password), u.PwSalt...))
}

func (u *User) Verify(username, password string) bool {
	if u.Name != username {
		panic(errors.New("invalid bbolt user entry: username does not match")) // TODO: log.
	}
	return cipher.SumSHA256(append([]byte(password), u.PwSalt...)) == u.PwHash
}

func (u *User) Encode() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(u); err != nil {
		panic(err) // TODO: log.
	}
	return buf.Bytes()
}

func DecodeBoltUser(raw []byte) *User {
	var user User
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&user); err != nil {
		panic(err) // TODO: log this.
	}
	return &user
}

type UserStorer interface {
	NewUser(username, password string) bool
	DeleteUser(username string)
	HasUser(username string) bool
	VerifyPassword(username, password string) bool
	ChangePassword(username, newPassword string) bool
}

type UsersConfig struct {
	DBPath          string
	SaltLen         int    // Salt Len for password verification data.
	UsernamePattern string // regular expression for usernames (no check if empty). TODO
	PasswordPattern string // regular expression for passwords (no check of empty). TODO
}

type BoltUserStore struct {
	db *bbolt.DB
	c  UsersConfig
}

func NewBoltUserStore(config UsersConfig) (*BoltUserStore, error) {
	if err := os.MkdirAll(filepath.Dir(config.DBPath), os.FileMode(0700)); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(config.DBPath, os.FileMode(0600), &bbolt.Options{Timeout: boltTimeout})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(boltUserBucketName))
		return err
	})
	return &BoltUserStore{db: db, c: config}, err
}

func (s *BoltUserStore) NewUser(username, password string) bool {
	var ok bool
	catch(s.db.Update(func(tx *bbolt.Tx) error {
		users := tx.Bucket([]byte(boltUserBucketName))
		if len(users.Get([]byte(username))) > 0 {
			ok = false
			return nil
		}
		user := User{Name: username}
		user.SetPassword(s.c.SaltLen, password)
		if err := users.Put([]byte(username), user.Encode()); err != nil {
			return err
		}
		ok = true
		return nil
	}))
	return ok
}

func (s *BoltUserStore) DeleteUser(username string) {
	catch(s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(boltUserBucketName)).Delete([]byte(username))
	}))
}

func (s *BoltUserStore) HasUser(username string) bool {
	var ok bool
	catch(s.db.View(func(tx *bbolt.Tx) error {
		ok = tx.Bucket([]byte(boltUserBucketName)).Get([]byte(username)) != nil
		return nil
	}))
	return ok
}

func (s *BoltUserStore) VerifyPassword(username, password string) bool {
	var ok bool
	catch(s.db.View(func(tx *bbolt.Tx) error {
		raw := tx.Bucket([]byte(boltUserBucketName)).Get([]byte(username))
		if len(raw) == 0 {
			ok = false
			return nil
		}
		ok = DecodeBoltUser(raw).Verify(username, password)
		return nil
	}))
	return ok
}

func (s *BoltUserStore) ChangePassword(username, newPassword string) bool {
	var ok bool
	catch(s.db.Update(func(tx *bbolt.Tx) error {
		users := tx.Bucket([]byte(boltUserBucketName))
		rawUser := users.Get([]byte(username))
		if len(rawUser) == 0 {
			ok = false
			return nil
		}
		user := DecodeBoltUser(rawUser)
		user.SetPassword(s.c.SaltLen, newPassword)
		if err := users.Put([]byte(boltUserBucketName), user.Encode()); err != nil {
			return err
		}
		ok = true
		return nil
	}))
	return ok
}
