package tunnel

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"

	"github.com/andrewmthomas87/web-p2p-tunnel/internal/signaling"
	"github.com/pion/webrtc/v4"
)

type Signaler interface {
	Offers() <-chan signaling.Offer
	Answers() chan<- signaling.Answer
	RemoteICECandidates() <-chan signaling.ICECandidate
	LocalICECandidates() chan<- signaling.ICECandidate
}

type Hub struct {
	log *log.Logger

	webrtcConfig webrtc.Configuration

	transport http.RoundTripper
	tunnels   map[string]*Tunnel
}

func NewHub(target *url.URL, changeHostHeader, changeOriginHeader bool, webrtcConfig webrtc.Configuration) *Hub {
	return &Hub{
		log:          log.New(os.Stderr, "[Tunnel hub] ", log.LstdFlags),
		webrtcConfig: webrtcConfig,
		transport:    newHandlerTransport(newSingleHostReverseProxy(target, changeHostHeader, changeOriginHeader)),
		tunnels:      make(map[string]*Tunnel),
	}
}

func (h *Hub) Run(ctx context.Context, signaler Signaler) error {
	h.log.Println("Running...")

	offers := signaler.Offers()
	answers := signaler.Answers()
	remoteICECandidates := signaler.RemoteICECandidates()
	localICECandidates := signaler.LocalICECandidates()

	for {
		select {
		case offer := <-offers:
			answer, err := h.handleOffer(offer, h.onICECandidate(offer.ClientID, localICECandidates))
			if err != nil {
				return err
			}
			answers <- answer

		case iceCandidate := <-remoteICECandidates:
			if err := h.handleRemoteICECandidate(iceCandidate); err != nil {
				return err
			}

		case <-ctx.Done():
			return h.close()

		}
	}
}

func (h *Hub) onICECandidate(clientID string, localICECandidates chan<- signaling.ICECandidate) func(*webrtc.ICECandidate) {
	return func(iceCandidate *webrtc.ICECandidate) {
		if iceCandidate == nil {
			return
		}

		localICECandidates <- signaling.ICECandidate{
			ClientID: clientID,
			Data:     iceCandidate.ToJSON(),
		}
	}
}

func (h *Hub) handleOffer(
	offer signaling.Offer,
	onICECandidate func(*webrtc.ICECandidate),
) (signaling.Answer, error) {
	_, ok := h.tunnels[offer.ClientID]
	if ok {
		return signaling.Answer{}, errors.New("received offer for open tunnel")
	}

	t, err := NewTunnel(h.webrtcConfig, h.transport, offer.ClientID, onICECandidate)
	if err != nil {
		return signaling.Answer{}, err
	}
	h.tunnels[offer.ClientID] = t

	h.log.Printf("Created tunnel for client %s", offer.ClientID)

	answer, err := t.RegisterOffer(offer.Data)
	if err != nil {
		return signaling.Answer{}, err
	}

	return signaling.Answer{
		ClientID: offer.ClientID,
		Data:     answer,
	}, nil
}

func (h *Hub) handleRemoteICECandidate(iceCandidate signaling.ICECandidate) error {
	t, ok := h.tunnels[iceCandidate.ClientID]
	if !ok {
		return errors.New("received message for unknown tunnel")
	}

	if err := t.AddICECandidate(iceCandidate.Data); err != nil {
		return err
	}

	return nil
}

func (h *Hub) close() error {
	h.log.Println("Closing tunnels...")

	for _, t := range h.tunnels {
		err := t.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

type handlerTransport struct {
	handler http.Handler
	http.ResponseWriter
}

func newHandlerTransport(handler http.Handler) http.RoundTripper {
	return &handlerTransport{handler: handler}
}

func (ht *handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rw := httptest.NewRecorder()
	ht.handler.ServeHTTP(rw, req)

	return rw.Result(), nil
}
