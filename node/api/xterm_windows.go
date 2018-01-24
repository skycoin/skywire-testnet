package api

import (
	"net/http"
	"os/exec"
	"github.com/gorilla/websocket"
	"fmt"
	log "github.com/sirupsen/logrus"
)

func xterm(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	cmd := exec.Command("cmd","/K","dir")
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	stdIn, err := cmd.StdinPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	go func() {
		defer func() {
			conn.Close()
		}()
		for {
			buf := make([]byte, 1024)
			read, err := stdOut.Read(buf)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()
	go func() {
		defer func() {
			conn.Close()
		}()
		for {
			_, reader, err := conn.ReadMessage()
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("System error: %s", err.Error())))
				return
			}
			log.Infof("read client: %v", reader)
			copied, err := stdIn.Write(reader)
			//copied, err := io.Copy(stdIn, reader)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error after copying %d bytes", copied)))
			}
			//dataTypeBuf := make([]byte, 1)
			//_, err = reader.Read(dataTypeBuf)
			//if err != nil {
			//	conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			//	return
			//}
			//switch dataTypeBuf[0] {
			//case 0:
			//
			//}
		}
	}()
	err = cmd.Start()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
}
