package client

import (
	"context"
	"fmt"

	"github.com/skycoin/dmsg/cipher"

	"github.com/skycoin/skywire/pkg/routing"
	"github.com/skycoin/skywire/pkg/transport"
)

// MockClient implements mock route finder client.
type mockClient struct {
	err error
}

// NewMock constructs a new mock Client.
func NewMock() Client {
	return &mockClient{}
}

// SetError assigns error that will be return on the next call to a
// public method.
func (r *mockClient) SetError(err error) {
	r.err = err
}

// FindRoutes implements Client for MockClient
func (r *mockClient) FindRoutes(ctx context.Context, rts []routing.PathEdges, opts *RouteOptions) (map[routing.PathEdges][]routing.Path, error) {
	if r.err != nil {
		return nil, r.err
	}

	if len(rts) == 0 {
		return nil, fmt.Errorf("no edges provided to returns routes from")
	}
	return map[routing.PathEdges][]routing.Path{
		[2]cipher.PubKey{rts[0][0], rts[0][1]}: {
			{
				routing.Hop{
					From:      rts[0][0],
					To:        rts[0][1],
					Transport: transport.MakeTransportID(rts[0][0], rts[0][1], ""),
				},
			},
		},
	}, nil
}
