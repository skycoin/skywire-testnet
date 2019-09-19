// Package client implements transport discovery client
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"

	"github.com/SkycoinProject/skywire-mainnet/internal/httpauth"
	"github.com/SkycoinProject/skywire-mainnet/pkg/transport"
)

var log = logging.MustGetLogger("transport-discovery")

// Error is the object returned to the client when there's an error.
type Error struct {
	Error string `json:"error"`
}

// apiClient implements Client for discovery API.
type apiClient struct {
	client *httpauth.Client
	key    cipher.PubKey
	sec    cipher.SecKey
}

// NewHTTP creates a new client setting a public key to the client to be used for auth.
// When keys are set, the client will sign request before submitting.
// The signature information is transmitted in the header using:
// * SW-Public: The specified public key
// * SW-Nonce:  The nonce for that public key
// * SW-Sig:    The signature of the payload + the nonce
func NewHTTP(addr string, key cipher.PubKey, sec cipher.SecKey) (transport.DiscoveryClient, error) {
	client, err := httpauth.NewClient(context.Background(), addr, key, sec)
	if err != nil {
		return nil, fmt.Errorf("httpauth: %s", err)
	}

	return &apiClient{client: client, key: key, sec: sec}, nil
}

// Post performs a POST request.
func (c *apiClient) Post(ctx context.Context, path string, payload interface{}) (*http.Response, error) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.client.Addr()+path, body)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req.WithContext(ctx))
}

// Get performs a new GET request.
func (c *apiClient) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.client.Addr()+path, new(bytes.Buffer))
	if err != nil {
		return nil, err
	}

	return c.client.Do(req.WithContext(ctx))
}

// RegisterTransports registers new Transports.
func (c *apiClient) RegisterTransports(ctx context.Context, entries ...*transport.SignedEntry) error {
	if len(entries) == 0 {
		return nil
	}

	resp, err := c.Post(ctx, "/transports/", entries)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close HTTP response body")
			}
		}()
	}
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	return fmt.Errorf("status: %d, error: %v", resp.StatusCode, extractError(resp.Body))
}

// GetTransportByID returns Transport for corresponding ID.
func (c *apiClient) GetTransportByID(ctx context.Context, id uuid.UUID) (*transport.EntryWithStatus, error) {
	resp, err := c.Get(ctx, fmt.Sprintf("/transports/id:%s", id.String()))
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close HTTP response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, error: %v", resp.StatusCode, extractError(resp.Body))
	}

	entry := &transport.EntryWithStatus{}
	if err := json.NewDecoder(resp.Body).Decode(entry); err != nil {
		return nil, fmt.Errorf("json: %s", err)
	}

	return entry, nil
}

// GetTransportsByEdge returns all Transports registered for the edge.
func (c *apiClient) GetTransportsByEdge(ctx context.Context, pk cipher.PubKey) ([]*transport.EntryWithStatus, error) {
	resp, err := c.Get(ctx, fmt.Sprintf("/transports/edge:%s", pk))
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close HTTP response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, error: %v", resp.StatusCode, extractError(resp.Body))
	}

	var entries []*transport.EntryWithStatus
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("json: %s", err)
	}

	return entries, nil
}

// UpdateStatuses updates statuses of transports in discovery.
func (c *apiClient) UpdateStatuses(ctx context.Context, statuses ...*transport.Status) ([]*transport.EntryWithStatus, error) {
	if len(statuses) == 0 {
		return nil, nil
	}

	resp, err := c.Post(ctx, "/statuses", statuses)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close HTTP response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, error: %v", resp.StatusCode, extractError(resp.Body))
	}

	var entries []*transport.EntryWithStatus
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("json: %s", err)
	}

	return entries, nil
}

// extractError returns the decoded error message from Body.
func extractError(r io.Reader) error {
	var apiError Error

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &apiError); err != nil {
		return errors.New(string(body))
	}

	return errors.New(apiError.Error)
}
