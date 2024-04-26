package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type Client struct {
	log *log.Logger

	RoomID string

	serverURL *url.URL

	offers              chan Offer
	answers             chan Answer
	remoteICECandidates chan ICECandidate
	localICECandidates  chan ICECandidate

	conn *websocket.Conn
	done chan struct{}
}

type Offer struct {
	ClientID string
	Data     webrtc.SessionDescription
}

type Answer struct {
	ClientID string
	Data     webrtc.SessionDescription
}

type ICECandidate struct {
	ClientID string
	Data     webrtc.ICECandidateInit
}

func NewClient(roomID string, serverURL *url.URL) *Client {
	return &Client{
		log:                 log.New(os.Stderr, fmt.Sprintf("[Signaling client %s] ", roomID[:6]), log.LstdFlags),
		RoomID:              roomID,
		serverURL:           serverURL,
		offers:              make(chan Offer, 16),
		answers:             make(chan Answer, 16),
		remoteICECandidates: make(chan ICECandidate, 16),
		localICECandidates:  make(chan ICECandidate, 16),
		done:                make(chan struct{}),
	}
}

func (c *Client) Offers() <-chan Offer {
	return c.offers
}

func (c *Client) Answers() chan<- Answer {
	return c.answers
}

func (c *Client) RemoteICECandidates() <-chan ICECandidate {
	return c.remoteICECandidates
}

func (c *Client) LocalICECandidates() chan<- ICECandidate {
	return c.localICECandidates
}

func (c *Client) Connect() error {
	c.log.Println("Connecting...")

	wsURL := c.serverURL.JoinPath("ws")

	query := wsURL.Query()
	query.Set("role", "server")
	query.Set("room-id", c.RoomID)
	wsURL.RawQuery = query.Encode()

	if c.serverURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	} else {
		wsURL.Scheme = "ws"
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		return err
	}
	c.conn = conn

	c.log.Println("Connected")

	return nil
}

func (c *Client) Run(ctx context.Context) error {
	c.log.Println("Running...")

	go c.readPump()
	go c.writePump()

	select {
	case <-c.done:
	case <-ctx.Done():
		c.log.Println("Closing...")

		err := c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		if err != nil {
			return err
		}

		select {
		case <-c.done:
		case <-time.After(time.Second):
			err := c.conn.Close()
			if err != nil {
				return err
			}
		}
	}

	c.log.Println("Closed")

	return nil
}

func (c *Client) readPump() {
	defer close(c.done)
	defer c.conn.Close()

	for {
		var message ServerMessage
		if err := c.conn.ReadJSON(&message); err != nil {
			return
		}

		switch message.Type {
		case "offer":
			c.log.Println("Received offer...")

			var data webrtc.SessionDescription
			if err := json.Unmarshal(message.Data, &data); err != nil {
				return
			}

			c.offers <- Offer{
				ClientID: message.ClientID,
				Data:     data,
			}

		case "icecandidate":
			c.log.Println("Received ICE candidate...")

			var data webrtc.ICECandidateInit
			if err := json.Unmarshal(message.Data, &data); err != nil {
				return
			}

			c.remoteICECandidates <- ICECandidate{
				ClientID: message.ClientID,
				Data:     data,
			}
		}
	}
}

func (c *Client) writePump() {
	for {
		select {
		case answer := <-c.answers:
			c.log.Println("Sending answer...")

			if err := c.writeMessage(answer.ClientID, "answer", answer.Data); err != nil {
				return
			}

		case iceCandidate := <-c.localICECandidates:
			c.log.Println("Sending ICE candidate...")

			if err := c.writeMessage(iceCandidate.ClientID, "icecandidate", iceCandidate.Data); err != nil {
				return
			}

		}
	}
}

func (c *Client) writeMessage(clientID string, messageType string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	message := ServerMessage{
		ClientID: clientID,
		Message: Message{
			Type: messageType,
			Data: b,
		},
	}
	if err := c.conn.WriteJSON(message); err != nil {
		return err
	}

	return nil
}
