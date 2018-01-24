package api

import (
	"net/http"
	"fmt"
	"os/exec"
	"github.com/gorilla/websocket"
	"io"
	"os"
	"github.com/kr/pty"
	"encoding/json"
	"syscall"
	"unsafe"
	log "github.com/sirupsen/logrus"
)

func xterm(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	cmd := exec.Command("/bin/bash")
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	go func() {
		defer func() {
			cmd.Process.Kill()
			cmd.Process.Wait()
			tty.Close()
			conn.Close()
		}()
		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	go func() {
		defer func() {
			cmd.Process.Kill()
			cmd.Process.Wait()
			tty.Close()
			conn.Close()
		}()
		for {

			_, reader, err := conn.NextReader()
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("System error: %s", err.Error())))
				return
			}
			dataTypeBuf := make([]byte, 1)
			log.Infof("dataTypeBuf: %v", dataTypeBuf)
			_, err = reader.Read(dataTypeBuf)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
				return
			}
			switch dataTypeBuf[0] {
			case 0:
				copied, err := io.Copy(tty, reader)
				if err != nil {
					conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error after copying %d bytes", copied)))
				}
			case 1:
				decoder := json.NewDecoder(reader)
				resizeMessage := windowSize{}
				err := decoder.Decode(&resizeMessage)
				if err != nil {
					conn.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
					continue
				}
				_, _, errno := syscall.Syscall(
					syscall.SYS_IOCTL,
					tty.Fd(),
					syscall.TIOCSWINSZ,
					uintptr(unsafe.Pointer(&resizeMessage)),
				)
				if errno != 0 {
					conn.WriteMessage(websocket.TextMessage, []byte("Unable to resize terminal: "+err.Error()))
				}
			}

		}
	}()
}
