package client_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skywire/pkg/cipher"

	"github.com/skycoin/skywire/pkg/messaging-discovery/client"
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

func TestNewClientEntryIsValid(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	cases := []struct {
		name  string
		entry func() *client.Entry
	}{
		{
			name: "NewClientEntry is valid",
			entry: func() *client.Entry {
				return client.NewClientEntry(pk, 0, nil)
			},
		},
		{
			name: "NewServerEntry is valid",
			entry: func() *client.Entry {
				return client.NewServerEntry(pk, 0, "localhost:8080", 5)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := tc.entry()
			err := entry.Sign(sk)
			require.NoError(t, err)

			err = entry.Validate()

			assert.NoError(t, err)
		})
	}
}

func TestVerifySignature(t *testing.T) {
	// Arrange
	// Create keys and signed entry
	pk, sk := cipher.GenerateKeyPair()
	wrongPk, _ := cipher.GenerateKeyPair()

	entry := newTestEntry(pk)

	err := entry.Sign(sk)
	require.NoError(t, err)

	// Action
	err = entry.VerifySignature()

	// Assert
	assert.Nil(t, err)

	// Action
	entry.Static = wrongPk
	err = entry.VerifySignature()

	// Assert
	assert.NotNilf(t, err, "this signature must not be valid")
}

func TestValidateRightEntry(t *testing.T) {
	// Arrange
	// Create keys and signed entry
	pk, sk := cipher.GenerateKeyPair()

	validEntry := newTestEntry(pk)
	err := validEntry.Sign(sk)
	require.NoError(t, err)

	// Action
	err = validEntry.Validate()
	assert.Nil(t, err)
}

func TestValidateNonKeysEntry(t *testing.T) {
	// Arrange
	// Create keys and signed entry
	_, sk := cipher.GenerateKeyPair()
	nonKeysEntry := client.Entry{
		Timestamp: time.Now().Unix(),
		Client:    &client.Client{},
		Server: &client.Server{
			Address:              "localhost:8080",
			AvailableConnections: 3,
		},
		Version:  "0",
		Sequence: 0,
	}
	err := nonKeysEntry.Sign(sk)
	require.NoError(t, err)

	// Action
	err = nonKeysEntry.Validate()
	assert.NotNil(t, err)
}

func TestValidateNonClientNonServerEntry(t *testing.T) {
	// Arrange
	// Create keys and signed entry
	_, sk := cipher.GenerateKeyPair()
	nonClientNonServerEntry := client.Entry{
		Timestamp: time.Now().Unix(),
		Version:   "0",
		Sequence:  0,
	}
	err := nonClientNonServerEntry.Sign(sk)
	require.NoError(t, err)

	// Action
	err = nonClientNonServerEntry.Validate()
	assert.NotNil(t, err)
}

func TestValidateNonSignedEntry(t *testing.T) {
	// Arrange
	// Create keys and signed entry
	nonClientNonServerEntry := client.Entry{
		Timestamp: time.Now().Unix(),
		Version:   "0",
		Sequence:  0,
	}

	// Action
	err := nonClientNonServerEntry.Validate()
	assert.NotNil(t, err)
}

func TestValidateIteration(t *testing.T) {
	// Arrange
	// Create keys and two entries
	pk, sk := cipher.GenerateKeyPair()

	entryPrevious := newTestEntry(pk)
	entryNext := newTestEntry(pk)
	entryNext.Sequence = 1
	err := entryPrevious.Sign(sk)
	require.NoError(t, err)

	// Action
	err = entryPrevious.ValidateIteration(&entryNext)

	// Assert
	assert.NoError(t, err)
}

func TestValidateIterationEmptyClient(t *testing.T) {
	// Arrange
	// Create keys and two entries
	pk, sk := cipher.GenerateKeyPair()

	entryPrevious := newTestEntry(pk)
	err := entryPrevious.Sign(sk)
	require.NoError(t, err)
	entryNext := newTestEntry(pk)
	entryNext.Sequence = 1
	err = entryNext.Sign(sk)
	require.NoError(t, err)

	// Action
	errValidation := entryNext.Validate()
	errIteration := entryPrevious.ValidateIteration(&entryNext)

	// Assert
	assert.NoError(t, errValidation)
	assert.NoError(t, errIteration)
}

func TestValidateIterationWrongSequence(t *testing.T) {
	// Arrange
	// Create keys and two entries
	pk, sk := cipher.GenerateKeyPair()

	entryPrevious := newTestEntry(pk)
	entryPrevious.Sequence = 2
	err := entryPrevious.Sign(sk)
	require.NoError(t, err)
	entryNext := newTestEntry(pk)
	err = entryNext.Sign(sk)
	require.NoError(t, err)

	// Action
	err = entryPrevious.ValidateIteration(&entryNext)

	// Assert
	assert.NotNil(t, err)
}

func TestValidateIterationWrongTime(t *testing.T) {
	// Arrange
	// Create keys and two entries
	pk, sk := cipher.GenerateKeyPair()

	entryPrevious := newTestEntry(pk)
	err := entryPrevious.Sign(sk)
	require.NoError(t, err)
	entryNext := newTestEntry(pk)
	entryNext.Timestamp -= 3
	err = entryNext.Sign(sk)
	require.NoError(t, err)

	// Action
	err = entryPrevious.ValidateIteration(&entryNext)

	// Assert
	assert.NotNil(t, err)
}

func TestCopy(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()
	entry := newTestEntry(pk)
	err := entry.Sign(sk)
	require.NoError(t, err)

	cases := []struct {
		name string
		src  *client.Entry
		dst  *client.Entry
	}{
		{
			name: "must copy values for client, server and keys",
			src:  &entry,
			dst: &client.Entry{
				Client:    &client.Client{},
				Server:    &client.Server{Address: "s", AvailableConnections: 0},
				Static:    cipher.PubKey{},
				Timestamp: 3,
				Sequence:  0,
				Version:   "0",
				Signature: "s",
			},
		},
		{
			name: "must accept dst empty entry",
			src:  &entry,
			dst:  &client.Entry{},
		},
		{
			name: "must accept src empty entry",
			src:  &client.Entry{},
			dst: &client.Entry{
				Client:    &client.Client{},
				Server:    &client.Server{Address: "s", AvailableConnections: 0},
				Static:    cipher.PubKey{},
				Timestamp: 3,
				Sequence:  0,
				Version:   "0",
				Signature: "s",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client.Copy(tc.dst, tc.src)

			assert.EqualValues(t, tc.src, tc.dst)
			if tc.dst.Server != nil {
				assert.NotEqual(t, fmt.Sprintf("%p", tc.dst.Server), fmt.Sprintf("%p", tc.src.Server))
			}
			if tc.dst.Client != nil {
				assert.NotEqual(t, fmt.Sprintf("%p", tc.dst.Client), fmt.Sprintf("%p", tc.src.Client))
			}
		})
	}
}
