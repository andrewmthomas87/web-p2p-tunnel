package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/andrewmthomas87/web-p2p-tunnel/internal/signaling"
	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":8080", "http server address")
	upgrader = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	flag.Parse()

	s := signaling.NewServer(upgrader)

	http.HandleFunc("/rooms", s.CreateRoomHandler)
	http.HandleFunc("/ws", s.WebSocketHandler)

	log.Printf("signaling-server starting at %s...", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
