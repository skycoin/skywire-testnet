package factory

import (
	"sync"

	"sync/atomic"

	"github.com/skycoin/skycoin/src/cipher"
)

func init() {
	ops[OP_QUERY_SERVICE_NODES] = &sync.Pool{
		New: func() interface{} {
			return new(query)
		},
	}
	resps[OP_QUERY_SERVICE_NODES] = &sync.Pool{
		New: func() interface{} {
			return new(QueryResp)
		},
	}

	ops[OP_QUERY_BY_ATTRS] = &sync.Pool{
		New: func() interface{} {
			return new(queryByAttrs)
		},
	}
	resps[OP_QUERY_BY_ATTRS] = &sync.Pool{
		New: func() interface{} {
			return new(QueryByAttrsResp)
		},
	}
}

var (
	querySeq uint32
)

type query struct {
	Keys []cipher.PubKey
	Seq  uint32
}

func newQuery(keys []cipher.PubKey) *query {
	q := &query{Keys: keys, Seq: atomic.AddUint32(&querySeq, 1)}
	return q
}

func (query *query) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if !f.Proxy {
		r = &QueryResp{
			Seq:    query.Seq,
			Result: f.findServiceAddresses(query.Keys, conn.GetKey()),
		}
		return
	}
	f.ForEachConn(func(connection *Connection) {
		connection.setProxyConnection(query.Seq, conn)
		connection.writeOP(OP_QUERY_SERVICE_NODES, query)
	})

	return
}

type QueryResp struct {
	Seq    uint32
	Result []*ServiceInfo
}

func (resp *QueryResp) Run(conn *Connection) (err error) {
	if connection, ok := conn.removeProxyConnection(resp.Seq); ok {
		return connection.writeOP(OP_QUERY_SERVICE_NODES|RESP_PREFIX, resp)
	}
	if conn.findServiceNodesByKeysCallback != nil {
		conn.findServiceNodesByKeysCallback(resp)
	}
	return
}

// query nodes by attributes
type queryByAttrs struct {
	Attrs []string
	Seq   uint32
	Pages int
	Limit int
}

func newQueryByAttrs(attrs []string) *queryByAttrs {
	q := &queryByAttrs{
		Attrs: attrs,
		Seq:   atomic.AddUint32(&querySeq, 1),
	}
	return q
}

func newQueryByAttrsAndPage(pages, limit int, attrs []string) *queryByAttrs {
	q := &queryByAttrs{
		Attrs: attrs,
		Pages: pages,
		Limit: limit,
		Seq:   atomic.AddUint32(&querySeq, 1),
	}
	return q
}

func (query *queryByAttrs) Execute(f *MessengerFactory, conn *Connection) (r resp, err error) {
	if query.Limit == 0 {
		query.Limit = 5
	}
	if !f.Proxy {
		r = &QueryByAttrsResp{Seq: query.Seq, Result: f.FindByAttributesAndPaging(query.Pages, query.Limit, query.Attrs...)}
		return
	}
	f.ForEachConn(func(connection *Connection) {
		connection.setProxyConnection(query.Seq, conn)
		connection.writeOP(OP_QUERY_BY_ATTRS, query)
	})

	return
}

type QueryByAttrsResp struct {
	Result *AttrNodesInfo
	Seq    uint32
}

func (resp *QueryByAttrsResp) Run(conn *Connection) (err error) {
	if connection, ok := conn.removeProxyConnection(resp.Seq); ok {
		return connection.writeOP(OP_QUERY_BY_ATTRS|RESP_PREFIX, resp)
	}
	if conn.findServiceNodesByAttributesCallback != nil {
		conn.findServiceNodesByAttributesCallback(resp)
	}
	return
}
