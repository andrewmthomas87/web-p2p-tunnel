package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pion/webrtc/v4"
)

type Tunnel struct {
	log *log.Logger

	pc *webrtc.PeerConnection
}

func NewTunnel(clientID string, onICECandidate func(*webrtc.ICECandidate)) (*Tunnel, error) {
	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}}},
	)
	if err != nil {
		return nil, err
	}

	t := &Tunnel{
		log: log.New(os.Stderr, fmt.Sprintf("[Tunnel %s] ", clientID[:6]), log.LstdFlags),
		pc:  pc,
	}

	pc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		t.log.Printf("Connection state change: %s", pcs)
	})
	pc.OnICECandidate(onICECandidate)
	pc.OnDataChannel(t.onDataChannel)

	return t, nil
}

func (t *Tunnel) Close() error {
	t.log.Println("Closing...")

	return t.pc.Close()
}

func (t *Tunnel) RegisterOffer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	t.log.Println("Registering offer...")

	if err := t.pc.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	t.log.Println("Creating answer...")

	answer, err := t.pc.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	if err := t.pc.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}

	return answer, nil
}

func (t *Tunnel) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	t.log.Println("Adding ICE candidate...")

	return t.pc.AddICECandidate(candidate)
}

func (t *Tunnel) onDataChannel(dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		t.log.Printf("Data channel opened: %s, %d", dc.Label(), *dc.ID())
	})
	dc.OnClose(func() {
		t.log.Printf("Data channel closed: %s, %d", dc.Label(), *dc.ID())
	})

	if dc.Label() == "http" {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(msg.Data)))
			if err != nil {
				t.log.Printf("Failed to read request: %v", err)
			}

			t.log.Printf("%s %s", req.Method, req.URL)

			if err := dc.Send(nil); err != nil {
				t.log.Printf("Failed to send response: %v", err)
			}
		})
	}
}
