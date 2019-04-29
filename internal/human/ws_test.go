package human

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/skycoin/skywire/internal/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsocketGrabber(t *testing.T) {
	message := "message"
	details := "details"
	opts := []UserOption{
		{
			Label: "red label",
			Color: color.Red,
		},
		{
			Label: "black label",
			Color: color.Black,
		},
	}
	requestJSON := "{\"message\":\"message\",\"details\":\"details\",\"opts\":[{\"label\":\"red label\",\"color\":\"red\"},{\"label\":\"black label\",\"color\":\"black\"}]}\n"
	response := "response"

	upgrader := &websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() {
				_ = c.Close()
			}()
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					break
				}

				assert.Equal(t, requestJSON, string(message))

				err = c.WriteJSON(wsResponsePayload{response})
				if err != nil {
					break
				}
			}
		},
	))
	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	grabber := NewWebsocketGrabber(conn)
	resp, err := grabber.GrabInput(context.Background(), message, details, opts)

	require.NoError(t, err)
	assert.Equal(t, response, resp)
}
