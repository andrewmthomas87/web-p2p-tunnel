package tunnel

import (
	"errors"
	"log"
	"os"

	"github.com/andrewmthomas87/web-p2p-tunnel/internal/signaling"
	"github.com/pion/webrtc/v4"
)

type Hub struct {
	log *log.Logger

	tunnels map[string]*Tunnel
}

func NewHub() *Hub {
	return &Hub{
		log:     log.New(os.Stderr, "[Tunnel hub] ", log.LstdFlags),
		tunnels: make(map[string]*Tunnel),
	}
}

func (h *Hub) Run(
	done <-chan struct{},
	offers <-chan signaling.Offer,
	answers chan<- signaling.Answer,
	remoteICECandidates <-chan signaling.ICECandidate,
	localICECandidates chan<- signaling.ICECandidate,
) error {
	h.log.Println("Running...")

	for {
		select {
		case offer := <-offers:
			_, ok := h.tunnels[offer.ClientID]
			if ok {
				return errors.New("received offer for open tunnel")
			}

			onICECandidate := func(iceCandidate *webrtc.ICECandidate) {
				if iceCandidate == nil {
					return
				}

				localICECandidates <- signaling.ICECandidate{
					ClientID: offer.ClientID,
					Data:     iceCandidate.ToJSON(),
				}
			}
			t, err := NewTunnel(offer.ClientID, onICECandidate)
			if err != nil {
				return err
			}
			h.tunnels[offer.ClientID] = t

			h.log.Printf("Created tunnel for client %s", offer.ClientID)

			answer, err := t.RegisterOffer(offer.Data)
			if err != nil {
				return err
			}

			answers <- signaling.Answer{
				ClientID: offer.ClientID,
				Data:     answer,
			}

		case iceCandidate := <-remoteICECandidates:
			t, ok := h.tunnels[iceCandidate.ClientID]
			if !ok {
				return errors.New("received message for unknown tunnel")
			}

			if err := t.AddICECandidate(iceCandidate.Data); err != nil {
				return err
			}

		case <-done:
			h.log.Println("Closing tunnels...")

			for _, t := range h.tunnels {
				err := t.Close()
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}
