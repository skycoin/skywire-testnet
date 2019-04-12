//go:generate esc -o static.go -prefix static static

/*
chat app for skywire node
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

	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
)

var addr = flag.String("addr", ":8000", "address to bind")

var (
	clientChan chan string
	chatConns  map[cipher.PubKey]net.Conn
	connsMu    sync.Mutex
)

func main() {
	app.Setup("chat", "1.0")
	defer app.Close()

	flag.Parse()

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
		conn, err := app.Accept()
		if err != nil {
			log.Println("failed to accept conn: ", err)
			return
		}

		raddr := conn.RemoteAddr().(*app.LoopAddr)
		connsMu.Lock()
		chatConns[raddr.PubKey] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	raddr := conn.RemoteAddr().(*app.LoopAddr)
	for {
		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("failed to read packet: ", err)
			return
		}

		clientMsg, _ := json.Marshal(map[string]string{"sender": raddr.PubKey.Hex(), "message": string(buf[:n])}) // nolint
		select {
		case clientChan <- string(clientMsg):
		default:
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

	addr := app.LoopAddr{PubKey: pk, Port: 1}
	connsMu.Lock()
	conn := chatConns[pk]
	connsMu.Unlock()

	if conn == nil {
		var err error
		conn, err = app.Dial(addr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		connsMu.Lock()
		chatConns[pk] = conn
		connsMu.Unlock()

		go handleConn(conn)
	}

	if _, err := conn.Write([]byte(data["message"])); err != nil {
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

	clientChan = make(chan string)
	go func() {
		<-req.Context().Done()
		close(clientChan)
		clientChan = nil
		log.Println("SSE connection were closed.")
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for msg := range clientChan {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		f.Flush()
	}
}
