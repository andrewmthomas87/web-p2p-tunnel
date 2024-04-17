package signaling

import (
	"io"
	"net/http"
	"net/url"
)

func CreateRoom(signalingServerURL *url.URL) (string, error) {
	createRoomURL := signalingServerURL.JoinPath("rooms")
	resp, err := http.Post(createRoomURL.String(), "", nil)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	return string(body), nil
}
