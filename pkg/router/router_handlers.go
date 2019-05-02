package router

import (
	"io"

	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/transport"
)

type handlerFunc func(ProcManager, io.ReadWriter) error

type routerHandlers struct {
	r        *router
	pm       ProcManager
	setupPKs map[cipher.PubKey]struct{}
}

func makeRouterHandlers(r *router, pm ProcManager) routerHandlers {
	setupPKs := make(map[cipher.PubKey]struct{})
	for _, pk := range r.conf.SetupNodes {
		setupPKs[pk] = struct{}{}
	}

	return routerHandlers{r, pm, setupPKs}
}

// determines if the given transport is established with a setup node.
func (rh routerHandlers) isSetup(tp transport.Transport) bool {
	pk, ok := rh.r.tpm.Remote(tp.Edges())
	if !ok {
		return false
	}
	_, ok = rh.setupPKs[pk]
	return ok
}

// serves a given transport with the 'handler' running in a loop.
// the loop exits on error.
func (rh routerHandlers) serve(tp transport.Transport, handle handlerFunc) error {
	for {
		if err := handle(rh.pm, tp); err != nil && err != io.EOF {
			rh.r.log.Warnf("Stopped serving Transport: %s", err)
			return err
		}
	}
}
