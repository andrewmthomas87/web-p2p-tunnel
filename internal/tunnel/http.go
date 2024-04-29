package tunnel

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/pion/webrtc/v4"
)

const mtu = 16*1024 - 1

type HTTPDataChannel struct {
	log *log.Logger

	client *http.Client
	dc     *webrtc.DataChannel

	r *io.PipeReader
	w *io.PipeWriter
}

func NewHTTPDataChannel(client *http.Client, dc *webrtc.DataChannel) *HTTPDataChannel {
	r, w := io.Pipe()

	h := &HTTPDataChannel{
		log:    log.New(os.Stderr, fmt.Sprintf("[HTTP Data Channel %d] ", *dc.ID()), log.LstdFlags),
		client: client,
		dc:     dc,
		r:      r,
		w:      w,
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
	req.RequestURI = ""

	h.log.Printf("%s %s", req.Method, req.URL)

	resp, err := h.client.Do(req)
	if err != nil {
		h.log.Printf("Proxied request failed: %v", err)

		resp = &http.Response{
			Status:     http.StatusText(http.StatusBadGateway),
			StatusCode: http.StatusBadGateway,
		}
	}

	if err := h.writeResponse(resp); err != nil {
		h.log.Printf("Failed to write response: %v", err)

		_ = h.dc.Close()
		return
	}
}

func (h *HTTPDataChannel) Write(p []byte) (n int, err error) {
	if len(p) > mtu {
		return 0, errors.New("data size exceeds MTU")
	}

	if err := h.dc.Send(p); err != nil {
		return 0, err
	}

	return len(p), nil
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

func (h *HTTPDataChannel) writeResponse(resp *http.Response) error {
	w := bufio.NewWriterSize(h, mtu)
	if err := resp.Write(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := h.dc.Send(nil); err != nil {
		return err
	}

	return nil
}
