package httpauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/skycoin/skywire/pkg/cipher"
)

// NextNonceResponse represents a ServeHTTP response for json encoding
type NextNonceResponse struct {
	Edge      cipher.PubKey `json:"edge"`
	NextNonce Nonce         `json:"next_nonce"`
}

// HTTPResponse represents the http response struct
type HTTPResponse struct {
	Error *HTTPError  `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// HTTPError is included in an HTTPResponse
type HTTPError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Client implements Client for auth services.
type Client struct {
	client http.Client
	key    cipher.PubKey
	sec    cipher.SecKey
	Addr   string // sanitized address of the client, which may differ from addr used in NewClient
	nonce  uint64
}

// NewClient creates a new client setting a public key to the client to be used for Auth.
// When keys are set, the client will sign request before submitting.
// The signature information is transmitted in the header using:
// * SW-Public: The specified public key
// * SW-Nonce:  The nonce for that public key
// * SW-Sig:    The signature of the payload + the nonce
func NewClient(ctx context.Context, addr string, key cipher.PubKey, sec cipher.SecKey) (*Client, error) {
	c := &Client{
		client: http.Client{},
		key:    key,
		sec:    sec,
		Addr:   sanitizedAddr(addr),
	}

	// request server for a nonce
	nonce, err := c.Nonce(ctx, c.key)
	if err != nil {
		return nil, err
	}
	c.nonce = uint64(nonce)

	return c, nil
}

// Do performs a new authenticated Request and returns the response. Internally, if the request was
// successful nonce is incremented
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("SW-Public", c.key.Hex())

	// use nonce, later, if no err from req update such nonce
	req.Header.Add("SW-Nonce", strconv.FormatUint(uint64(c.getCurrentNonce()), 10))

	body := make([]byte, 0)
	if req.ContentLength != 0 {
		auxBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(auxBody))
		body = auxBody
	}
	sign, err := Sign(body, c.getCurrentNonce(), c.sec)
	if err != nil {
		return nil, err
	}

	req.Header.Add("SW-Sig", sign.Hex())

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusOK {
		c.incrementNonce()
	}

	return res, nil
}

// Nonce calls the remote API to retrieve the next expected nonce
func (c *Client) Nonce(ctx context.Context, key cipher.PubKey) (Nonce, error) {
	req, err := http.NewRequest(http.MethodGet, c.Addr+"/security/nonces/"+key.Hex(), nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("error getting current nonce: status: %d <- %v", resp.StatusCode, extractError(resp.Body))
	}

	var nr NextNonceResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		return 0, err
	}

	return Nonce(nr.NextNonce), nil
}

// SetNonce sets client current nonce to given nonce
func (c *Client) SetNonce(n Nonce) {
	atomic.StoreUint64(&c.nonce, uint64(n))
}

func (c *Client) getCurrentNonce() Nonce {
	return Nonce(atomic.LoadUint64(&c.nonce))
}

func (c *Client) incrementNonce() {
	atomic.AddUint64(&c.nonce, 1)
}

func sanitizedAddr(addr string) string {
	if addr == "" {
		return "http://localhost"
	}

	u, err := url.Parse(addr)
	if err != nil {
		return "http://localhost"
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

// extractError returns the decoded error message from Body.
func extractError(r io.Reader) error {
	var serverError HTTPResponse

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &serverError); err != nil {
		return errors.New(string(body))
	}

	return errors.New(serverError.Error.Message)
}
