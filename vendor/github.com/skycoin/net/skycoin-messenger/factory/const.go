package factory

import "github.com/skycoin/skycoin/src/cipher"

const (
	MSG_OP_SIZE         = 1
	MSG_PUBLIC_KEY_SIZE = 33
)

const (
	MSG_HEADER_BEGIN = 0
	MSG_OP_BEGIN
	MSG_OP_END = MSG_OP_BEGIN + MSG_OP_SIZE
	MSG_HEADER_END
)

const (
	SEND_MSG_META_BEGIN = MSG_HEADER_END

	SEND_MSG_PUBLIC_KEY_BEGIN
	SEND_MSG_PUBLIC_KEY_END = SEND_MSG_PUBLIC_KEY_BEGIN + MSG_PUBLIC_KEY_SIZE

	SEND_MSG_TO_PUBLIC_KEY_BEGIN
	SEND_MSG_TO_PUBLIC_KEY_END = SEND_MSG_TO_PUBLIC_KEY_BEGIN + MSG_PUBLIC_KEY_SIZE

	SEND_MSG_META_END
)

const (
	// request public key for the connection
	OP_REG = iota
	// im messages
	OP_SEND
	// app custom messages
	OP_CUSTOM
	// discovery register
	OP_OFFER_SERVICE
	// find services by public key (cxo)
	OP_QUERY_SERVICE_NODES
	// find services by attributes (vpn, socks etc)
	OP_QUERY_BY_ATTRS

	// build udp p2p connections
	OP_BUILD_APP_CONN
	OP_FORWARD_NODE_CONN
	OP_BUILD_NODE_CONN
	OP_FORWARD_NODE_CONN_RESP
	OP_BUILD_APP_CONN_OK
	OP_APP_CONN_ACK
	OP_APP_FEEDBACK

	// reg with key steps
	OP_REG_KEY
	OP_REG_SIG

	OP_SIZE
)

const RESP_PREFIX = 0x80

var EMPATY_PUBLIC_KEY = cipher.PubKey{}
