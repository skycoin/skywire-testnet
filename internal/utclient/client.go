// Package utclient implements uptime tracker client
package utclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/skycoin/skywire/internal/httpauth"
	"github.com/skycoin/skywire/pkg/cipher"
)

// Error is the object returned to the client when there's an error.
type Error struct {
	Error string `json:"error"`
}

// APIClient implements messaging discovery API client.
type APIClient interface {
	UpdateNodeUptime(context.Context) error
}

// httpClient implements Client for uptime tracker API.
type httpClient struct {
	client *httpauth.Client
	pk     cipher.PubKey
	sk     cipher.SecKey
}

// NewHTTP creates a new client setting a public key to the client to be used for auth.
// When keys are set, the client will sign request before submitting.
// The signature information is transmitted in the header using:
// * SW-Public: The specified public key
// * SW-Nonce:  The nonce for that public key
// * SW-Sig:    The signature of the payload + the nonce
func NewHTTP(addr string, pk cipher.PubKey, sk cipher.SecKey) (APIClient, error) {
	client, err := httpauth.NewClient(context.Background(), addr, pk, sk)
	if err != nil {
		return nil, fmt.Errorf("httpauth: %s", err)
	}

	return &httpClient{client: client, pk: pk, sk: sk}, nil
}

// Get performs a new GET request.
func (c *httpClient) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.client.Addr+path, new(bytes.Buffer))
	if err != nil {
		return nil, err
	}

	return c.client.Do(req.WithContext(ctx))
}

// UpdateNodeUptime updates node uptime.
func (c *httpClient) UpdateNodeUptime(ctx context.Context) error {
	resp, err := c.Get(ctx, "/update")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %d, error: %v", resp.StatusCode, extractError(resp.Body))
	}

	return nil
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
