package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/pion/webrtc/v4"
)

const mtu = 16*1024 - 1

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
	dc.OnOpen(func() {
		t.log.Printf("Data channel opened: %s, %d", dc.Label(), *dc.ID())
	})
	dc.OnClose(func() {
		t.log.Printf("Data channel closed: %s, %d", dc.Label(), *dc.ID())
	})

	if dc.Label() == "http" {
		dc.OnMessage(t.handleHTTP(dc))
	}
}

func (t *Tunnel) handleHTTP(dc *webrtc.DataChannel) func(msg webrtc.DataChannelMessage) {
	var b bytes.Buffer

	respond := func() {
		req, err := http.ReadRequest(bufio.NewReader(&b))
		if err != nil {
			t.log.Printf("Failed to read request: %v", err)

			_ = dc.Close()
			return
		}
		req.RequestURI = ""

		t.log.Printf("%s %s", req.Method, req.URL)

		if req.URL.Scheme == t.originURL.Scheme && req.URL.Host == t.originURL.Host {
			req.URL.Scheme = t.targetURL.Scheme
			req.URL.Host = t.targetURL.Host
		}

		resp, err := t.client.Do(req)
		if err != nil {
			t.log.Printf("Proxied request failed: %v", err)

			resp = &http.Response{
				Status:     http.StatusText(http.StatusBadGateway),
				StatusCode: http.StatusBadGateway,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
			}
		}

		header, err := httputil.DumpResponse(resp, false)
		if err != nil {
			t.log.Printf("Failed to dump response: %v", err)

			_ = dc.Close()
			return
		}

		var b bytes.Buffer
		b.Write(header)

		if resp.Body != nil {
			if _, err := io.Copy(&b, resp.Body); err != nil {
				t.log.Printf("Failed to build response: %v", err)

				_ = dc.Close()
				return
			}
		}

		out := b.Bytes()

		count := len(out) / mtu
		if len(out)%mtu > 0 {
			count++
		}
		for i := 0; i < count; i++ {
			f := out[i*mtu : min((i+1)*mtu, len(out))]
			if err := dc.Send(f); err != nil {
				t.log.Printf("Failed to send response: %v", err)

				_ = dc.Close()
				return
			}
		}

		if err := dc.Send(nil); err != nil {
			t.log.Printf("Failed to send response: %v", err)

			_ = dc.Close()
			return
		}
	}

	return func(msg webrtc.DataChannelMessage) {
		if len(msg.Data) == 0 {
			respond()
			return
		}

		if _, err := b.Write(msg.Data); err != nil {
			t.log.Printf("Failed to write message data: %v", err)

			_ = dc.Close()
			return
		}
	}
}
