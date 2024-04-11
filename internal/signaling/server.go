package signaling

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Server struct {
	log *log.Logger

	upgrader *websocket.Upgrader

	rooms     map[string]*Room
	roomsLock sync.RWMutex
}

func NewServer(upgrader *websocket.Upgrader) *Server {
	return &Server{
		log:      log.New(os.Stderr, "[Server] ", log.LstdFlags),
		upgrader: upgrader,
		rooms:    make(map[string]*Room),
	}
}

func (s *Server) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	id := uuid.NewString()
	room := NewRoom(id)

	s.roomsLock.Lock()
	s.rooms[id] = room
	s.roomsLock.Unlock()

	s.log.Printf("Created room %s", id)

	if _, err := w.Write([]byte(id)); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	roomID := r.URL.Query().Get("room-id")
	if !((role == "client" || role == "server") && roomID != "") {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	s.roomsLock.RLock()
	room, ok := s.rooms[roomID]
	s.roomsLock.RUnlock()
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.log.Printf("Adding %s ws conn to room %s...", role, room.ID)

	if role == "client" {
		room.HandleClientConn(conn)
	} else {
		room.HandleServerConn(conn)
	}
}
