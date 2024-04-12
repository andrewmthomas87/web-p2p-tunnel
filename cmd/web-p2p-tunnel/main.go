package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var (
	signalingServerURLStr = flag.String("signaling-server-url", "http://localhost:8080", "signaling server url")
)

func main() {
	flag.Parse()

	signalingServerURL, err := url.Parse(*signalingServerURLStr)
	if err != nil {
		log.Fatal(err)
	}

	createRoomURL := signalingServerURL.JoinPath("rooms")
	resp, err := http.Post(createRoomURL.String(), "", nil)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	roomID := string(body)

	log.Printf("Created room %s", roomID)

	wsURL := signalingServerURL.JoinPath("ws")

	query := wsURL.Query()
	query.Set("role", "server")
	query.Set("room-id", roomID)
	wsURL.RawQuery = query.Encode()

	if signalingServerURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	} else {
		wsURL.Scheme = "ws"
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		log.Fatal(resp, err)
	}
	defer conn.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			log.Printf("Recv: %s, type: %d", message, messageType)
		}
	}()

	select {
	case <-done:
	case <-interrupt:
		log.Println("signal: interrupt")

		err := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			return
		}

		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
