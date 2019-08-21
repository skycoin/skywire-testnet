//go:generate esc -o static.go -prefix static static

/*
skychat app for skywire visor
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/skywire/internal/netutil"
	th "github.com/skycoin/skywire/internal/testhelpers"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/routing"
)

var (
	addr   = flag.String("addr", ":8000", "address to bind")
	logger = logging.NewMasterLogger().PackageLogger("chat")
	r      = netutil.NewRetrier(50*time.Millisecond, 0, 2)

	chatApp   *app.App
	clientCh  chan string
	chatConns map[cipher.PubKey]net.Conn
	connsMu   sync.Mutex

	trcLog = logger.WithField("_module", th.GetCallerN(4))
)

func trStart() error { // nolint:unparam
	logger.Debug(th.Trace("ENTER"))
	return nil
}
func trFinish(_ error) { logger.Debug(th.Trace("EXIT")) }

func main() {
	flag.Parse()

	a, err := app.Setup(&app.Config{AppName: "skychat", AppVersion: "1.0", ProtocolVersion: "0.0.1"})
	if err != nil {
		trcLog.Fatal("Setup failure", err)
	}
	defer func() {
		if err := a.Close(); err != nil {
			trcLog.Info("Failed to close app:", err)
		}
	}()

	chatApp = a

	clientCh = make(chan string)
	defer close(clientCh)

	chatConns = make(map[cipher.PubKey]net.Conn)
	go listenLoop()

	http.Handle("/", http.FileServer(FS(false)))
	http.HandleFunc("/message", messageHandler)
	http.HandleFunc("/sse", sseHandler)

	trcLog.Info("Serving HTTP on", *addr)
	trcLog.Info(http.ListenAndServe(*addr, nil))
}

func listenLoop() {
	defer trFinish(trStart())

	for {
		conn, err := chatApp.Accept()
		if err != nil {
			log.Println("failed to accept conn:", err)
			return
		}

		raddr := conn.RemoteAddr().(routing.Addr)
		connsMu.Lock()
		chatConns[raddr.PubKey] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer trFinish(trStart())
	var cntr uint64
	raddr := conn.RemoteAddr().(routing.Addr)

	for {
		atomic.AddUint64(&cntr, 1)

		trcLog.Debugf("CYCLE %03d START", cntr)
		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil {
			trcLog.Debug("failed to read packet:", err)
			return
		}

		clientMsg, err := json.Marshal(map[string]string{"sender": raddr.PubKey.Hex(), "message": string(buf[:n])})
		if err != nil {
			trcLog.Debug("Failed to marshal json: ", err)
		}
		select {
		case clientCh <- string(clientMsg):
			trcLog.Debugf("received and sent to ui: %s\n", clientMsg)
		default:
			trcLog.Debugf("received and trashed: %s\n", clientMsg)
		}
		trcLog.Debugf("CYCLE %03d END", cntr)
	}
}

func messageHandler(w http.ResponseWriter, req *http.Request) {
	defer trFinish(trStart())

	data := map[string]string{}
	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pk := cipher.PubKey{}
	if err := pk.UnmarshalText([]byte(data["recipient"])); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	addr := routing.Addr{PubKey: pk, Port: 1}
	trcLog.Debug("addr: ", addr)

	connsMu.Lock()
	conn, ok := chatConns[pk]
	connsMu.Unlock()
	trcLog.Debugf("chatConn: %v  pk:%v\n", chatConns, pk)

	var cntr uint64

	if !ok {
		var err error
		err = r.Do(func() error {
			atomic.AddUint64(&cntr, 1)
			trcLog.Debugf("dial %v  addr:%v\n", cntr, addr)
			conn, err = chatApp.Dial(addr)
			return err
		})
		if err != nil {
			trcLog.Debug("err: ", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		connsMu.Lock()
		chatConns[pk] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}

	_, err := conn.Write([]byte(data["message"]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		connsMu.Lock()
		delete(chatConns, pk)
		connsMu.Unlock()
		return
	}

}

func sseHandler(w http.ResponseWriter, req *http.Request) {
	defer trFinish(trStart())

	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for {
		select {
		case msg, ok := <-clientCh:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", msg)
			f.Flush()

		case <-req.Context().Done():
			log.Println("SSE connection were closed.")
			return
		}
	}
}
