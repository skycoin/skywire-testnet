package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/skycoin/skywire/internal/httputil"
	"github.com/skycoin/skywire/pkg/cipher"
	"go.etcd.io/bbolt"
	"net/http"
	"os"
	"path/filepath"
)

type AuthErrorType string

const (
	UnknownType            = AuthErrorType("Unknown")
	UserAlreadyExistsType  = AuthErrorType("UserAlreadyExists")
	VerificationFailedType = AuthErrorType("VerificationFailed")
)

type authError struct {
	Type       AuthErrorType
	HTTPStatus    int
	HTTPMsg       string
	LogMsg        string
}

func (e authError) Error() string {
	if e.LogMsg != "" {
		return e.LogMsg
	}
	return e.HTTPMsg
}

func (e authError) WriteHTTP(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, r, e.HTTPStatus, errors.New(e.HTTPMsg))
}

func AuthError(err error) *authError {
	authErr, ok := err.(*authError)
	if !ok {
		authErr = &authError{
			Type:       UnknownType,
			HTTPStatus: http.StatusInternalServerError,
			HTTPMsg:    "unknown error",
			LogMsg:     err.Error(),
		}
	}
	return authErr
}

func ErrUserAlreadyExists(username string) error {
	return &authError{
		Type:       UserAlreadyExistsType,
		HTTPStatus: http.StatusForbidden,
		HTTPMsg:    fmt.Sprintf("user of name '%s' already exists", username),
	}
}

func ErrVerificationFailed(username string, userExists bool) error {
	var logMsg string
	if userExists {
		logMsg = fmt.Sprintf("verification failed: invalid password provided for existing user '%s'", username)
	} else {
		logMsg = fmt.Sprintf("verification failed: no such user of username '%s'", username)
	}
	return &authError{
		Type:       VerificationFailedType,
		HTTPStatus: http.StatusUnauthorized,
		HTTPMsg:    "username or password incorrect",
		LogMsg:     logMsg,
	}
}

type UserEntry struct {
	Name   string        `json:"name"`
	PwSalt []byte        `json:"pw_salt"`
	PwHash cipher.SHA256 `json:"pw_hash"`
}

func (u *UserEntry) SetPassword(saltLen int, password string) {
	u.PwSalt = cipher.RandByte(saltLen)
	u.PwHash = cipher.SumSHA256(append([]byte(password), u.PwSalt...))
}

func (u *UserEntry) Verify(username, password string) error {
	if u.Name != username {
		return errors.New("invalid bbolt user entry: username does not match")
	}
	if cipher.SumSHA256(append([]byte(password), u.PwSalt...)) != u.PwHash {
		return ErrVerificationFailed(username, true)
	}
	return nil
}

func (u *UserEntry) Encode() []byte {
	raw, err := json.Marshal(u)
	if err != nil {
		panic(err) // TODO: log this.
	}
	return raw
}

func (u *UserEntry) Decode(username string, raw []byte) {
	if err := json.Unmarshal(raw, u); err != nil {
		panic(err) // TODO: log this.
	}
}

type AuthStorer interface {
	NewUser(username, password string) error
	DeleteUser(username string) error
	VerifyUser(username, password string) error
	ChangePassword(username, password string) error

	NewSession(username string) (*http.Cookie, error)
	DeleteSession(value string) error
	UserSessions(username string) ([]*http.Cookie, error)
}

type BoltAuthStore struct {
	db      *bbolt.DB
	saltLen int
}

func NewBoltAuthStore(path string, saltLen int) (*BoltAuthStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0700)); err != nil {
		return nil, err
	}
	db, err := bbolt.Open(path, os.FileMode(0600), &bbolt.Options{Timeout: boltTimeout})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(boltUserBucketName)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(boltSessionBucketName)); err != nil {
			return err
		}
		return nil
	})
	return &BoltAuthStore{db: db, saltLen: saltLen}, err
}

func (s *BoltAuthStore) NewUser(username, password string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(boltUserBucketName))
		if len(b.Get([]byte(username))) > 0 {
			return ErrUserAlreadyExists(username)
		}
		entry := UserEntry{Name: username}
		entry.SetPassword(s.saltLen, password)
		return b.Put([]byte(username), entry.Encode())
	})
}

func (s *BoltAuthStore) DeleteUser(username string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.
			Bucket([]byte(boltUserBucketName)).
			Delete([]byte(username))
	})
}

func (s *BoltAuthStore) VerifyUser(username, password string) error {
	return s.db.View(func(tx *bbolt.Tx) error {
		raw := tx.
			Bucket([]byte(boltUserBucketName)).
			Get([]byte(username))
		if len(raw) == 0 {
			return ErrVerificationFailed(username, false)
		}
		var entry UserEntry
		entry.Decode(username, raw)
		return entry.Verify(username, password)
	})
}

func (s *BoltAuthStore) ChangePassword(username, password string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(boltUserBucketName))
		raw := b.Get([]byte(username))
		if len(raw) == 0 {
			return ErrVerificationFailed(username, false) // TODO: Change
		}
		var entry UserEntry
		entry.Decode(username, raw)
		entry.SetPassword(s.saltLen, password)
		return b.Put([]byte(boltUserBucketName), entry.Encode())
	})
}