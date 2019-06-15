//go:generate esc -o static.go -prefix static static

/*
skychat app for skywire node
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
	"time"

	"github.com/skycoin/skywire/internal/netutil"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

var addr = flag.String("addr", ":8000", "address to bind")
var r = netutil.NewRetrier(50*time.Millisecond, time.Second, 2)

var (
	chatApp   *app.App
	clientCh  chan string
	chatConns map[cipher.PubKey]net.Conn
	connsMu   sync.Mutex
)

func main() {
	flag.Parse()

	a, err := app.Setup(&app.Config{AppName: "skychat", AppVersion: "1.0", ProtocolVersion: "0.0.1"})
	if err != nil {
		log.Fatal("Setup failure: ", err)
	}
	defer func() { _ = a.Close() }()

	chatApp = a

	clientCh = make(chan string)
	defer close(clientCh)

	chatConns = make(map[cipher.PubKey]net.Conn)
	go listenLoop()

	http.Handle("/", http.FileServer(FS(false)))
	http.HandleFunc("/message", messageHandler)
	http.HandleFunc("/sse", sseHandler)

	log.Println("Serving HTTP on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func listenLoop() {
	for {
		conn, err := chatApp.Accept()
		if err != nil {
			log.Println("failed to accept conn: ", err)
			return
		}

		raddr := conn.RemoteAddr().(*app.Addr)
		connsMu.Lock()
		chatConns[raddr.PubKey] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	raddr := conn.RemoteAddr().(*app.Addr)
	for {
		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("failed to read packet: ", err)
		}

		clientMsg, _ := json.Marshal(map[string]string{"sender": raddr.PubKey.Hex(), "message": string(buf[:n])}) // nolint
		select {
		case clientCh <- string(clientMsg):
			log.Printf("received and sent to ui: %s\n", clientMsg)
		default:
			log.Printf("received and trashed: %s\n", clientMsg)
		}
	}
}

func messageHandler(w http.ResponseWriter, req *http.Request) {
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

	addr := &app.Addr{PubKey: pk, Port: 1}
	connsMu.Lock()
	conn := chatConns[pk]
	connsMu.Unlock()

	if conn == nil {
		var err error
		conn, err = chatApp.Dial(addr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		connsMu.Lock()
		chatConns[pk] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}

	err := r.Do(func() error {
		_, err := conn.Write([]byte(data["message"]))
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func sseHandler(w http.ResponseWriter, req *http.Request) {
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
