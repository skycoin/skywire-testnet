package messages

import "github.com/skycoin/skycoin/src/cipher"

type NodeAppMessage struct {
	Sequence uint32
	AppId    AppId
	Payload  []byte
}

type NodeAppResponse struct {
	Sequence uint32
	Misc     []byte
}

type SendFromAppMessage struct {
	ConnectionId ConnectionId
	Payload      []byte
}

type RegisterAppMessage struct {
	AppType string
}

type AssignConnectionNAM struct {
	ConnectionId ConnectionId
}

type ConnectToAppMessage struct {
	Address cipher.PubKey
	AppFrom AppId
	AppTo   AppId
}

type AppListRequest struct {
	RequestType  string // "by_name", "by_type", "all"
	RequestParam string // type or service name, if all then equals ""
}

type AppListResponse struct {
	Apps []ServiceInfo
}
