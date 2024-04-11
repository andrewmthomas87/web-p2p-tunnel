package signaling

import "encoding/json"

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type ServerMessage struct {
	Message

	ClientID string `json:"clientID"`
}
