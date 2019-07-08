// Package client implement client for route finder.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
)

const defaultContextTimeout = 10 * time.Second

// GetRoutesRequest parses json body for /routes endpoint request
type GetRoutesRequest struct {
	SrcPK   cipher.PubKey `json:"src_pk,omitempty"`
	DstPK   cipher.PubKey `json:"dst_pk,omitempty"`
	MinHops uint16        `json:"min_hops,omitempty"`
	MaxHops uint16        `json:"max_hops,omitempty"`
}

// GetRoutesResponse encodes the json body of /routes response
type GetRoutesResponse struct {
	Forward []routing.Route `json:"forward"`
	Reverse []routing.Route `json:"response"`
}

// HTTPResponse represents http response struct
type HTTPResponse struct {
	Error *HTTPError  `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// HTTPError is included in an HTTPResponse
type HTTPError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Client implements route finding operations.
type Client interface {
	PairedRoutes(source, destiny cipher.PubKey, minHops, maxHops uint16) ([]routing.Route, []routing.Route, error)
}

// APIClient implements Client interface
type apiClient struct {
	addr       string
	client     http.Client
	apiTimeout time.Duration
}

// NewHTTP constructs new Client that communicates over http.
func NewHTTP(addr string, apiTimeout time.Duration) Client {
	if apiTimeout == 0 {
		apiTimeout = defaultContextTimeout
	}

	return &apiClient{
		addr:       sanitizedAddr(addr),
		client:     http.Client{},
		apiTimeout: apiTimeout,
	}
}

// PairedRoutes returns routes from source skywire node to destiny, that has at least the given minHops and as much
// the given maxHops as well as the reverse routes from destiny to source.
func (c *apiClient) PairedRoutes(source, destiny cipher.PubKey, minHops, maxHops uint16) ([]routing.Route, []routing.Route, error) {
	requestBody := &GetRoutesRequest{
		SrcPK:   source,
		DstPK:   destiny,
		MinHops: minHops,
		MaxHops: maxHops,
	}
	marshaledBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(http.MethodGet, c.addr+"/routes", bytes.NewBuffer(marshaledBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), c.apiTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != http.StatusOK {
		var apiErr HTTPResponse

		err = json.NewDecoder(res.Body).Decode(&apiErr)
		if err != nil {
			return nil, nil, err
		}
		defer res.Body.Close()

		return nil, nil, errors.New(apiErr.Error.Message)
	}

	var routes GetRoutesResponse
	err = json.NewDecoder(res.Body).Decode(&routes)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	return routes.Forward, routes.Reverse, nil
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
