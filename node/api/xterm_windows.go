package api

import (
	"net/http"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"fmt"
	"time"
)

func xterm(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	if err != nil {
		log.Infof("conn upgrade: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	conn.WriteMessage(websocket.TextMessage, []byte("windows unsupported"))
	return
}
