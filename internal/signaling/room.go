package signaling

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Room struct {
	log *log.Logger

	ID string

	serverConn      *websocket.Conn
	serverConnLock  sync.Mutex
	clientConns     map[string]*websocket.Conn
	clientConnsLock sync.Mutex
}

func NewRoom(id string) *Room {
	return &Room{
		log:         log.New(os.Stderr, fmt.Sprintf("[Room %s] ", id[:6]), log.LstdFlags),
		ID:          id,
		clientConns: make(map[string]*websocket.Conn),
	}
}

func (r *Room) HandleServerConn(conn *websocket.Conn) {
	r.serverConnLock.Lock()
	if r.serverConn != nil {
		conn.WriteJSON(Message{
			Type: "error",
			Data: json.RawMessage("server conn already exists"),
		})
		conn.Close()

		r.serverConnLock.Unlock()

		r.log.Printf("Rejected server conn: %v. Server conn already exists.", conn.RemoteAddr())

		return
	}

	r.serverConn = conn
	r.serverConnLock.Unlock()

	r.log.Printf("Registered server conn: %v", conn.RemoteAddr())

	defer func() {
		r.serverConnLock.Lock()
		r.serverConn = nil
		r.serverConnLock.Unlock()

		conn.Close()

		r.log.Printf("Server conn closed: %v", conn.RemoteAddr())
	}()

	for {
		var message ServerMessage
		if err := conn.ReadJSON(&message); err != nil {
			return
		}

		if err := r.sendMessageToClient(message.ClientID, message.Data); err != nil {
			return
		}
	}
}

func (r *Room) HandleClientConn(conn *websocket.Conn) {
	r.clientConnsLock.Lock()

	id := uuid.NewString()
	r.clientConns[id] = conn

	r.clientConnsLock.Unlock()

	r.log.Printf("Registered client conn: %v, %s", conn.RemoteAddr(), id)

	defer func() {
		r.clientConnsLock.Lock()
		delete(r.clientConns, id)
		r.clientConnsLock.Unlock()

		conn.Close()

		r.log.Printf("Client conn closed: %v, %s", conn.RemoteAddr(), id)
	}()

	for {
		var data json.RawMessage
		if err := conn.ReadJSON(&data); err != nil {
			return
		}

		if err := r.sendMessageToServer(id, data); err != nil {
			return
		}
	}
}

func (r *Room) sendMessageToClient(clientID string, data json.RawMessage) error {
	r.clientConnsLock.Lock()
	defer r.clientConnsLock.Unlock()

	conn, ok := r.clientConns[clientID]
	if !ok {
		return errors.New("unknown client")
	}

	if err := conn.WriteJSON(data); err != nil {
		return err
	}

	return nil
}

func (r *Room) sendMessageToServer(clientID string, data json.RawMessage) error {
	r.serverConnLock.Lock()
	defer r.serverConnLock.Unlock()

	if r.serverConn == nil {
		return errors.New("no server")
	}

	message := ServerMessage{
		Message: Message{
			Type: "client",
			Data: data,
		},
		ClientID: clientID,
	}
	if err := r.serverConn.WriteJSON(message); err != nil {
		return err
	}

	return nil
}
