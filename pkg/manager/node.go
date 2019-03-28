// Package manager implements management node
package manager

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"

	"github.com/skycoin/skywire/internal/httputil"
	"github.com/skycoin/skywire/internal/noise"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/node"
	"github.com/skycoin/skywire/pkg/routing"
)

var (
	log = logging.MustGetLogger("manager")
)

// Node manages AppNodes.
type Node struct {
	c     Config
	nodes map[cipher.PubKey]node.RPCClient // connected remote nodes.
	users *UserManager
	mu    *sync.RWMutex
}

// NewNode creates a new Node.
func NewNode(config Config) (*Node, error) {
	boltUserDB, err := NewBoltUserStore(config.DBPath)
	if err != nil {
		return nil, err
	}
	singleUserDB := NewSingleUserStore("admin", boltUserDB)

	return &Node{
		c:     config,
		nodes: make(map[cipher.PubKey]node.RPCClient),
		users: NewUserManager(singleUserDB, config.Cookies),
		mu:    new(sync.RWMutex),
	}, nil
}

// ServeRPC serves RPC of a Node.
func (m *Node) ServeRPC(lis net.Listener) error {
	for {
		conn, err := noise.WrapListener(lis, m.c.PK, m.c.SK, false, noise.HandshakeXK).Accept()
		if err != nil {
			return err
		}
		addr := conn.RemoteAddr().(*noise.Addr)
		m.mu.RLock()
		m.nodes[addr.PK] = node.NewRPCClient(rpc.NewClient(conn), node.RPCPrefix)
		m.mu.RUnlock()
	}
}

// MockConfig configures how mock data is to be added.
type MockConfig struct {
	Nodes            int
	MaxTpsPerNode    int
	MaxRoutesPerNode int
}

// AddMockData adds mock data to Manager Node.
func (m *Node) AddMockData(config MockConfig) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < config.Nodes; i++ {
		pk, client := node.NewMockRPCClient(r, config.MaxTpsPerNode, config.MaxRoutesPerNode)
		m.mu.Lock()
		m.nodes[pk] = client
		m.mu.Unlock()
	}
	return nil
}

// ServeHTTP implements http.Handler
func (m *Node) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r := chi.NewRouter()

	r.Use(middleware.Timeout(time.Second * 30))
	r.Use(middleware.Logger)

	r.Route("/api", func(r chi.Router) {

		r.Group(func(r chi.Router) {
			r.Post("/create-account", m.users.CreateAccount())
			r.Post("/login", m.users.Login())
			r.Post("/logout", m.users.Logout())
		})

		r.Group(func(r chi.Router) {
			r.Use(m.users.Authorize)

			r.Get("/user", m.users.UserInfo())
			r.Post("/change-password", m.users.ChangePassword())

			r.Get("/nodes", m.getNodes())
			r.Get("/nodes/{pk}", m.getNode())

			r.Get("/nodes/{pk}/apps", m.getApps())
			r.Get("/nodes/{pk}/apps/{app}", m.getApp())
			r.Put("/nodes/{pk}/apps/{app}", m.putApp())

			r.Get("/nodes/{pk}/transport-types", m.getTransportTypes())

			r.Get("/nodes/{pk}/transports", m.getTransports())
			r.Post("/nodes/{pk}/transports", m.postTransport())
			r.Get("/nodes/{pk}/transports/{tid}", m.getTransport())
			r.Delete("/nodes/{pk}/transports/{tid}", m.deleteTransport())

			r.Get("/nodes/{pk}/routes", m.getRoutes())
			r.Post("/nodes/{pk}/routes", m.postRoute())
			r.Get("/nodes/{pk}/routes/{rid}", m.getRoute())
			r.Put("/nodes/{pk}/routes/{rid}", m.putRoute())
			r.Delete("/nodes/{pk}/routes/{rid}", m.deleteRoute())
		})
	})

	r.ServeHTTP(w, req)
}

// provides summary of all nodes.
func (m *Node) getNodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var summaries []*node.Summary
		m.mu.RLock()
		for pk, c := range m.nodes {
			summary, err := c.Summary()
			if err != nil {
				log.Printf("failed to obtain summary from AppNode with pk %s. Error: %v", pk, err)
				summary = &node.Summary{PubKey: pk}
			}
			summaries = append(summaries, summary)
		}
		m.mu.RUnlock()
		httputil.WriteJSON(w, r, http.StatusOK, summaries)
	}
}

// provides summary of single node.
func (m *Node) getNode() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		summary, err := ctx.RPC.Summary()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, summary)
	})
}

// returns app summaries of a given node of pk
func (m *Node) getApps() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		apps, err := ctx.RPC.Apps()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, apps)
	})
}

// returns an app summary of a given node's pk and app name
func (m *Node) getApp() http.HandlerFunc {
	return m.ctxApp(func(w http.ResponseWriter, r *http.Request, ctx appCtx) {
		httputil.WriteJSON(w, r, http.StatusOK, ctx.App)
	})
}

func (m *Node) putApp() http.HandlerFunc {
	return m.ctxApp(func(w http.ResponseWriter, r *http.Request, ctx appCtx) {
		var reqBody struct {
			Autostart *bool `json:"autostart,omitempty"`
			Status    *int  `json:"status,omitempty"`
		}
		if err := httputil.ReadJSON(r, &reqBody); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		if reqBody.Autostart != nil {
			if *reqBody.Autostart != ctx.App.AutoStart {
				if err := ctx.RPC.SetAutoStart(ctx.App.Name, *reqBody.Autostart); err != nil {
					httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
					return
				}
			}
		}
		if reqBody.Status != nil {
			if *reqBody.Status == 0 {
				if err := ctx.RPC.StopApp(ctx.App.Name); err != nil {
					httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
					return
				}
			} else if *reqBody.Status == 1 {
				if err := ctx.RPC.StartApp(ctx.App.Name); err != nil {
					httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
					return
				}
			} else {
				httputil.WriteJSON(w, r, http.StatusBadRequest,
					fmt.Errorf("value of 'status' field is %d when expecting 0 or 1", *reqBody.Status))
				return
			}
		}
		httputil.WriteJSON(w, r, http.StatusOK, ctx.App)
	})
}

func (m *Node) getTransportTypes() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		types, err := ctx.RPC.TransportTypes()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, types)
	})
}

func (m *Node) getTransports() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		var (
			qTypes []string
			qPKs   []cipher.PubKey
			qLogs  bool
		)
		var err error
		qTypes = strSliceFromQuery(r, "type", nil)
		if qPKs, err = pkSliceFromQuery(r, "pk", nil); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		if qLogs, err = httputil.BoolFromQuery(r, "logs", true); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		transports, err := ctx.RPC.Transports(qTypes, qPKs, qLogs)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, transports)
	})
}

func (m *Node) postTransport() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		var reqBody struct {
			Remote cipher.PubKey `json:"remote_pk"`
			TpType string        `json:"transport_type"`
			Public bool          `json:"public"`
		}
		if err := httputil.ReadJSON(r, &reqBody); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		summary, err := ctx.RPC.AddTransport(reqBody.Remote, reqBody.TpType, reqBody.Public, 30*time.Second)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, summary)
	})
}

func (m *Node) getTransport() http.HandlerFunc {
	return m.ctxTransport(func(w http.ResponseWriter, r *http.Request, ctx transportCtx) {
		httputil.WriteJSON(w, r, http.StatusOK, ctx.Tp)
	})
}

func (m *Node) deleteTransport() http.HandlerFunc {
	return m.ctxTransport(func(w http.ResponseWriter, r *http.Request, ctx transportCtx) {
		if err := ctx.RPC.RemoveTransport(ctx.Tp.ID); err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	})
}

type routingRuleResp struct {
	Key     routing.RouteID      `json:"key"`
	Rule    string               `json:"rule"`
	Summary *routing.RuleSummary `json:"rule_summary,omitempty"`
}

func makeRoutingRuleResp(key routing.RouteID, rule routing.Rule, summary bool) routingRuleResp {
	resp := routingRuleResp{
		Key:  key,
		Rule: hex.EncodeToString(rule),
	}
	if summary {
		resp.Summary = rule.Summary()
	}
	return resp
}

func (m *Node) getRoutes() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		qSummary, err := httputil.BoolFromQuery(r, "summary", false)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		rules, err := ctx.RPC.RoutingRules()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		resp := make([]routingRuleResp, len(rules))
		for i, rule := range rules {
			resp[i] = makeRoutingRuleResp(rule.Key, rule.Value, qSummary)
		}
		httputil.WriteJSON(w, r, http.StatusOK, resp)
	})
}

func (m *Node) postRoute() http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		var summary routing.RuleSummary
		if err := httputil.ReadJSON(r, &summary); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		rule, err := summary.ToRule()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		tid, err := ctx.RPC.AddRoutingRule(rule)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, makeRoutingRuleResp(tid, rule, true))
	})
}

func (m *Node) getRoute() http.HandlerFunc {
	return m.ctxRoute(func(w http.ResponseWriter, r *http.Request, ctx routeCtx) {
		qSummary, err := httputil.BoolFromQuery(r, "summary", true)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		rule, err := ctx.RPC.RoutingRule(ctx.RtKey)
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusNotFound, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, makeRoutingRuleResp(ctx.RtKey, rule, qSummary))
	})
}

func (m *Node) putRoute() http.HandlerFunc {
	return m.ctxRoute(func(w http.ResponseWriter, r *http.Request, ctx routeCtx) {
		var summary routing.RuleSummary
		if err := httputil.ReadJSON(r, &summary); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		rule, err := summary.ToRule()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		if err := ctx.RPC.SetRoutingRule(ctx.RtKey, rule); err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, makeRoutingRuleResp(ctx.RtKey, rule, true))
	})
}

func (m *Node) deleteRoute() http.HandlerFunc {
	return m.ctxRoute(func(w http.ResponseWriter, r *http.Request, ctx routeCtx) {
		if err := ctx.RPC.RemoveRoutingRule(ctx.RtKey); err != nil {
			httputil.WriteJSON(w, r, http.StatusNotFound, err)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	})
}

/*
	<<< Helper functions >>>
*/

func (m *Node) client(pk cipher.PubKey) (node.RPCClient, bool) {
	m.mu.RLock()
	client, ok := m.nodes[pk]
	m.mu.RUnlock()
	return client, ok
}

func pkFromParam(r *http.Request, key string) (cipher.PubKey, error) {
	pk := cipher.PubKey{}
	err := pk.UnmarshalText([]byte(chi.URLParam(r, key)))
	return cipher.PubKey(pk), err
}

func uuidFromParam(r *http.Request, key string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, key))
}

func ridFromParam(r *http.Request, key string) (routing.RouteID, error) {
	rid, err := strconv.ParseUint(chi.URLParam(r, key), 10, 32)
	if err != nil {
		return 0, errors.New("invalid route ID provided")
	}
	return routing.RouteID(rid), nil
}

func strSliceFromQuery(r *http.Request, key string, defaultVal []string) []string {
	slice, ok := r.URL.Query()[key]
	if !ok {
		return defaultVal
	}
	return slice
}

func pkSliceFromQuery(r *http.Request, key string, defaultVal []cipher.PubKey) ([]cipher.PubKey, error) {
	qPKs, ok := r.URL.Query()[key]
	if !ok {
		return defaultVal, nil
	}
	pks := make([]cipher.PubKey, len(qPKs))
	for i, qPK := range qPKs {
		pk := cipher.PubKey{}
		if err := pk.UnmarshalText([]byte(qPK)); err != nil {
			return nil, err
		}
		pks[i] = cipher.PubKey(pk)
	}
	return pks, nil
}

type nodeCtx struct {
	PK  cipher.PubKey
	RPC node.RPCClient
}

type nodeHandlerFunc func(w http.ResponseWriter, r *http.Request, ctx nodeCtx)

func (m *Node) ctxNode(next nodeHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pk, err := pkFromParam(r, "pk")
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		client, ok := m.client(pk)
		if !ok {
			httputil.WriteJSON(w, r, http.StatusNotFound, fmt.Errorf("node of pk '%s' not found", pk))
			return
		}
		next(w, r, nodeCtx{PK: pk, RPC: client})
	}
}

type appCtx struct {
	nodeCtx
	App *node.AppState
}

type appHandlerFunc func(w http.ResponseWriter, r *http.Request, ctx appCtx)

func (m *Node) ctxApp(next appHandlerFunc) http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		appName := chi.URLParam(r, "app")
		apps, err := ctx.RPC.Apps()
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		for _, app := range apps {
			if app.Name == appName {
				next(w, r, appCtx{nodeCtx: ctx, App: app})
				return
			}
		}
		httputil.WriteJSON(w, r, http.StatusNotFound,
			fmt.Errorf("can not find app of name %s from node %s", appName, ctx.PK))
	})
}

type transportCtx struct {
	nodeCtx
	Tp *node.TransportSummary
}

type transportHandlerFunc func(w http.ResponseWriter, r *http.Request, ctx transportCtx)

func (m *Node) ctxTransport(next transportHandlerFunc) http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		tid, err := uuidFromParam(r, "tid")
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		tp, err := ctx.RPC.Transport(tid)
		if err != nil {
			if err.Error() == node.ErrNotFound.Error() {
				httputil.WriteJSON(w, r, http.StatusNotFound,
					fmt.Errorf("transport of ID %s is not found", tid))
				return
			}
			httputil.WriteJSON(w, r, http.StatusInternalServerError, err)
			return
		}
		next(w, r, transportCtx{nodeCtx: ctx, Tp: tp})
	})
}

type routeCtx struct {
	nodeCtx
	RtKey routing.RouteID
}

type routeHandlerFunc func(w http.ResponseWriter, r *http.Request, ctx routeCtx)

func (m *Node) ctxRoute(next routeHandlerFunc) http.HandlerFunc {
	return m.ctxNode(func(w http.ResponseWriter, r *http.Request, ctx nodeCtx) {
		rid, err := ridFromParam(r, "key")
		if err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
		}
		next(w, r, routeCtx{nodeCtx: ctx, RtKey: rid})
	})
}

func catch(err error, msgs ...string) {
	if err != nil {
		if len(msgs) > 0 {
			log.Fatalln(append(msgs, err.Error()))
		} else {
			log.Fatalln(err)
		}
	}
}
