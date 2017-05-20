package messages

const (
	SOCKS_CLIENT = iota
	SOCKS_SERVER
	VPN_CLIENT
	VPN_SERVER
)

var AppTypes []string

func init() {
	AppTypes[SOCKS_CLIENT] = "Socks client (listens to user)"
	AppTypes[SOCKS_SERVER] = "Socks server (listens to web)"
	AppTypes[VPN_CLIENT] = "VPN client (listens to user)"
	AppTypes[VPN_SERVER] = "VPN server (listens to web)"
}
