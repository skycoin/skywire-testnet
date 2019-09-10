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
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

const defaultContextTimeout = 10 * time.Second

var log = logging.MustGetLogger("route-finder")

type RouteOptions struct {
	MinHops uint16
	MaxHops uint16
}

// GetRoutesRequest parses json body for /routes endpoint request
type FindRoutesRequest struct {
	Edges [][2]cipher.PubKey
	Opts  *RouteOptions
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
	FindRoutes(ctx context.Context, rts [][2]cipher.PubKey, opts *RouteOptions) ([][]routing.Route, error)
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

// FindRoutes returns routes from source skywire visor to destiny, that has at least the given minHops and as much
// the given maxHops.
func (c *apiClient) FindRoutes(ctx context.Context, rts [][2]cipher.PubKey, opts *RouteOptions) ([][]routing.Route, error) {
	requestBody := &FindRoutesRequest{
		Edges: rts,
		Opts:  opts,
	}
	marshaledBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, c.addr+"/routes", bytes.NewBuffer(marshaledBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), c.apiTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if res != nil {
		defer func() {
			if err := res.Body.Close(); err != nil {
				log.WithError(err).Warn("Failed to close HTTP response body")
			}
		}()
	}
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		var apiErr HTTPResponse

		err = json.NewDecoder(res.Body).Decode(&apiErr)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(apiErr.Error.Message)
	}

	var routes [][]routing.Route
	err = json.NewDecoder(res.Body).Decode(&routes)
	if err != nil {
		return nil, err
	}

	return routes, nil
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
