package tunnel

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/pion/webrtc/v4"
)

type Tunnel struct {
	log *log.Logger

	originURL *url.URL
	targetURL *url.URL

	client *http.Client
	pc     *webrtc.PeerConnection
}

func NewTunnel(
	originURL,
	targetURL *url.URL,
	webrtcConfig webrtc.Configuration,
	clientID string,
	onICECandidate func(*webrtc.ICECandidate),
) (*Tunnel, error) {
	pc, err := webrtc.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, err
	}

	t := &Tunnel{
		log:       log.New(os.Stderr, fmt.Sprintf("[Tunnel %s] ", clientID[:6]), log.LstdFlags),
		originURL: originURL,
		targetURL: targetURL,
		client:    &http.Client{},
		pc:        pc,
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

	t.client.CloseIdleConnections()
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
	t.log.Printf("Data Channel %s, %d", dc.Label(), *dc.ID())

	if dc.Label() == "http" {
		hdc := NewHTTPDataChannel(t.originURL, t.targetURL, t.client, dc)
		go hdc.Run()
	}
}
