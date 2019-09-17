// Package disc implements client for dmsg discovery.
package disc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/SkycoinProject/SkycoinProject/src/util/logging"

	"github.com/SkycoinProject/dmsg/cipher"
)

var log = logging.MustGetLogger("disc")

// APIClient implements messaging discovery API client.
type APIClient interface {
	Entry(context.Context, cipher.PubKey) (*Entry, error)
	SetEntry(context.Context, *Entry) error
	UpdateEntry(context.Context, cipher.SecKey, *Entry) error
	AvailableServers(context.Context) ([]*Entry, error)
}

// HTTPClient represents a client that communicates with a messaging-discovery service through http, it
// implements APIClient
type httpClient struct {
	client    http.Client
	address   string
	updateMux sync.Mutex // for thread-safe sequence incrementing
}

// NewHTTP constructs a new APIClient that communicates with discovery via http.
func NewHTTP(address string) APIClient {
	return &httpClient{
		client:  http.Client{},
		address: address,
	}
}

// Entry retrieves an entry associated with the given public key.
func (c *httpClient) Entry(ctx context.Context, publicKey cipher.PubKey) (*Entry, error) {
	var entry Entry
	endpoint := fmt.Sprintf("%s/messaging-discovery/entry/%s", c.address, publicKey)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	// if the response is an error it will be codified as an HTTPMessage
	if resp.StatusCode != http.StatusOK {
		var message HTTPMessage
		err = json.NewDecoder(resp.Body).Decode(&message)
		if err != nil {
			return nil, err
		}

		return nil, errFromString(message.Message)
	}

	err = json.NewDecoder(resp.Body).Decode(&entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// SetEntry creates a new Entry.
func (c *httpClient) SetEntry(ctx context.Context, e *Entry) error {
	endpoint := c.address + "/messaging-discovery/entry/"
	marshaledEntry, err := json.Marshal(e)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(marshaledEntry))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close response body")
			}
		}()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var httpResponse HTTPMessage

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(bodyBytes, &httpResponse)
		if err != nil {
			return err
		}
		return errFromString(httpResponse.Message)
	}
	return nil
}

// UpdateEntry updates Entry in messaging discovery.
func (c *httpClient) UpdateEntry(ctx context.Context, sk cipher.SecKey, e *Entry) error {
	c.updateMux.Lock()
	defer c.updateMux.Unlock()

	e.Sequence++
	e.Timestamp = time.Now().UnixNano()

	for {
		err := e.Sign(sk)
		if err != nil {
			return err
		}
		err = c.SetEntry(ctx, e)
		if err == nil {
			return nil
		}
		if err != ErrValidationWrongSequence {
			e.Sequence--
			return err
		}
		rE, entryErr := c.Entry(ctx, e.Static)
		if entryErr != nil {
			return err
		}
		if rE.Timestamp > e.Timestamp { // If there is a more up to date entry drop update
			e.Sequence = rE.Sequence
			return nil
		}
		e.Sequence = rE.Sequence + 1
	}
}

// AvailableServers returns list of available servers.
func (c *httpClient) AvailableServers(ctx context.Context) ([]*Entry, error) {
	var entries []*Entry
	endpoint := c.address + "/messaging-discovery/available_servers"

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	// if the response is an error it will be codified as an HTTPMessage
	if resp.StatusCode != http.StatusOK {
		var message HTTPMessage
		err = json.NewDecoder(resp.Body).Decode(&message)
		if err != nil {
			return nil, err
		}

		return nil, errFromString(message.Message)
	}

	err = json.NewDecoder(resp.Body).Decode(&entries)
	if err != nil {
		return nil, err
	}

	return entries, nil
}
