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

type HTTPDataChannel struct {
	log *log.Logger

	originURL *url.URL
	targetURL *url.URL

	client *http.Client
	dc     *webrtc.DataChannel

	r *io.PipeReader
	w *io.PipeWriter
}

func NewHTTPDataChannel(
	originURL,
	targetURL *url.URL,
	client *http.Client,
	dc *webrtc.DataChannel,
) *HTTPDataChannel {
	r, w := io.Pipe()

	h := &HTTPDataChannel{
		log:       log.New(os.Stderr, fmt.Sprintf("[HTTP Data Channel %d] ", *dc.ID()), log.LstdFlags),
		originURL: originURL,
		targetURL: targetURL,
		client:    client,
		dc:        dc,
		r:         r,
		w:         w,
	}

	dc.OnMessage(h.onMessage)
	dc.OnClose(h.onClose)

	return h
}

func (h *HTTPDataChannel) Run() {
	req, err := http.ReadRequest(bufio.NewReader(h.r))
	if err != nil {
		h.log.Printf("Failed to read request: %v", err)

		_ = h.dc.Close()
		return
	}

	h.log.Printf("%s %s", req.Method, req.URL)

	req.RequestURI = ""
	if req.URL.Scheme == h.originURL.Scheme && req.URL.Host == h.originURL.Host {
		req.URL.Scheme = h.targetURL.Scheme
		req.URL.Host = h.targetURL.Host
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.log.Printf("Proxied request failed: %v", err)

		resp = &http.Response{
			Status:     http.StatusText(http.StatusBadGateway),
			StatusCode: http.StatusBadGateway,
		}
	}

	header, err := httputil.DumpResponse(resp, false)
	if err != nil {
		h.log.Printf("Failed to dump response: %v", err)

		_ = h.dc.Close()
		return
	}

	var b bytes.Buffer

	if _, err := b.Write(header); err != nil {
		h.log.Printf("Failed to build response: %v", err)

		_ = h.dc.Close()
		return
	}

	if resp.Body != nil {
		if _, err := io.Copy(&b, resp.Body); err != nil {
			h.log.Printf("Failed to build response: %v", err)

			_ = h.dc.Close()
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
		if err := h.dc.Send(f); err != nil {
			h.log.Printf("Failed to send response: %v", err)

			_ = h.dc.Close()
			return
		}
	}

	if err := h.dc.Send(nil); err != nil {
		h.log.Printf("Failed to send response: %v", err)

		_ = h.dc.Close()
		return
	}
}

func (h *HTTPDataChannel) onMessage(msg webrtc.DataChannelMessage) {
	if len(msg.Data) == 0 {
		_ = h.w.Close()
		return
	}

	if _, err := h.w.Write(msg.Data); err != nil {
		h.log.Printf("Failed to write message data: %v", err)

		_ = h.dc.Close()
		return
	}
}

func (h *HTTPDataChannel) onClose() {
	_ = h.w.Close()
}
