package tunnel

import (
	"fmt"
	"log"
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
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		t.log.Printf("Data channel: %s, %d", dc.Label(), *dc.ID())

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			t.log.Printf("Message: %s", msg.Data)
		})
	})

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
