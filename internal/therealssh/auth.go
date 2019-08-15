package therealssh

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/skycoin/dmsg/cipher"
)

// Authorizer defines interface for authorization providers.
type Authorizer interface {
	Authorize(pk cipher.PubKey) error
}

// ListAuthorizer performs authorization against static list.
type ListAuthorizer struct {
	AuthorizedKeys []cipher.PubKey
}

// Authorize implements Authorizer for ListAuthorizer
func (auth *ListAuthorizer) Authorize(remotePK cipher.PubKey) error {
	for _, key := range auth.AuthorizedKeys {
		if remotePK == key {
			return nil
		}
	}

	return errors.New("unknown PubKey")
}

// FileAuthorizer performs authorization against file that has PubKey per line.
type FileAuthorizer struct {
	authFile *os.File
}

// NewFileAuthorizer constructs new FileAuthorizer
func NewFileAuthorizer(authFile string) (*FileAuthorizer, error) {
	path, err := filepath.Abs(authFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth file path: %s", err)
	}

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return nil, fmt.Errorf("failed to create auth file: %s", err)
			}
			if f, err = os.Create(path); err != nil {
				return nil, fmt.Errorf("failed to create auth file: %s", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open auth file: %s", err)
		}
	}

	return &FileAuthorizer{f}, nil
}

// Close releases underlying file pointer.
func (auth *FileAuthorizer) Close() error {
	if auth == nil {
		return nil
	}
	return auth.authFile.Close()
}

// Authorize implements Authorizer for FileAuthorizer
func (auth *FileAuthorizer) Authorize(remotePK cipher.PubKey) error {
	defer func() {
		if _, err := auth.authFile.Seek(0, 0); err != nil {
			Logger.WithError(err).Warn("Failed to seek to the beginning of auth file")
		}
	}()

	hexPK := remotePK.Hex()
	scanner := bufio.NewScanner(auth.authFile)
	for scanner.Scan() {
		if hexPK == scanner.Text() {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read from auth file: %s", err)
	}

	return errors.New("unknown PubKey")
}
