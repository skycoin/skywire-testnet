package routerclient

import (
	"context"
	"fmt"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/pkg/routing"
)

// AddEdgeRules is a wrapper for (*Client).AddEdgeRules.
func AddEdgeRules(ctx context.Context, log *logging.Logger, dmsgC *dmsg.Client, pk cipher.PubKey, rules routing.EdgeRules) (bool, error) {
	client, err := NewClient(ctx, dmsgC, pk)
	if err != nil {
		return false, fmt.Errorf("failed to dial remote: %v", err)
	}
	defer closeClient(log, client)

	ok, err := client.AddEdgeRules(ctx, rules)
	if err != nil {
		return false, fmt.Errorf("failed to add rules: %v", err)
	}

	return ok, nil
}

// AddIntermediaryRules is a wrapper for (*Client).AddIntermediaryRules.
func AddIntermediaryRules(ctx context.Context, log *logging.Logger, dmsgC *dmsg.Client, pk cipher.PubKey, rules []routing.Rule) (bool, error) {
	client, err := NewClient(ctx, dmsgC, pk)
	if err != nil {
		return false, fmt.Errorf("failed to dial remote: %v", err)
	}
	defer closeClient(log, client)

	routeIDs, err := client.AddIntermediaryRules(ctx, rules)
	if err != nil {
		return false, fmt.Errorf("failed to add rules: %v", err)
	}

	return routeIDs, nil
}

// ReserveIDs is a wrapper for (*Client).ReserveIDs.
func ReserveIDs(ctx context.Context, log *logging.Logger, dmsgC *dmsg.Client, pk cipher.PubKey, n uint8) ([]routing.RouteID, error) {
	client, err := NewClient(ctx, dmsgC, pk)
	if err != nil {
		return nil, fmt.Errorf("failed to dial remote: %v", err)
	}
	defer closeClient(log, client)

	routeIDs, err := client.ReserveIDs(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("failed to add rules: %v", err)
	}

	return routeIDs, nil
}

func closeClient(log *logging.Logger, client *Client) {
	if err := client.Close(); err != nil {
		log.Warn(err)
	}
}
