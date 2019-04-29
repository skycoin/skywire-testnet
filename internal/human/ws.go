package human

import (
	"context"

	"github.com/gorilla/websocket"
)

// WebsocketGrabber requests for user input via an http interface exposing websocket connections.
type WebsocketGrabber struct {
	conn *websocket.Conn
}

// NewWebsocketGrabber creates a new WebsocketGrabber.
func NewWebsocketGrabber(conn *websocket.Conn) *WebsocketGrabber {
	return &WebsocketGrabber{
		conn: conn,
	}
}

type wsRequestPayload struct {
	Message string       `json:"message"`
	Details string       `json:"details"`
	Opts    []UserOption `json:"opts"`
}

type wsResponsePayload struct {
	Result string `json:"result"`
}

// GrabInput obtains the users input.
func (g *WebsocketGrabber) GrabInput(ctx context.Context, msg, details string, opts []UserOption) (string, error) {
	payload := wsRequestPayload{
		Message: msg,
		Details: details,
		Opts:    opts,
	}
	if err := g.conn.WriteJSON(payload); err != nil {
		return "", err
	}

	var resp wsResponsePayload
	if err := g.conn.ReadJSON(&resp); err != nil {
		return "", err
	}

	return resp.Result, nil
}
