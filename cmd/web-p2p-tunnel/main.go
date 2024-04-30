package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/andrewmthomas87/web-p2p-tunnel/internal/signaling"
	"github.com/andrewmthomas87/web-p2p-tunnel/internal/tunnel"
	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
)

var (
	signalingServerURLStr = flag.String("signaling-server-url", "http://localhost:8080", "signaling server url")
	tunnelTargetURLStr    = flag.String("tunnel-target-url", "", "tunnel target url")
	changeHostHeader      = flag.Bool(
		"change-host-header",
		false,
		"change the Host header to the host of the target url",
	)
	changeOriginHeader = flag.Bool(
		"change-origin-header",
		false,
		"change the Origin header's scheme & host to the scheme & host of the target url",
	)

	defaultWebrtcConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}
)

func main() {
	flag.Parse()

	signalingServerURL, err := url.Parse(*signalingServerURLStr)
	if err != nil {
		log.Fatal(err)
	}

	tunnelTargetURL, err := url.Parse(*tunnelTargetURLStr)
	if err != nil {
		log.Fatal(err)
	}

	roomID, err := signaling.CreateRoom(signalingServerURL)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Created room %s", roomID)

	sc := signaling.NewClient(roomID, signalingServerURL)
	if err := sc.Connect(); err != nil {
		log.Fatal(err)
	}

	th := tunnel.NewHub(tunnelTargetURL, *changeHostHeader, *changeOriginHeader, defaultWebrtcConfig)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return sc.Run(ctx)
	})
	g.Go(func() error {
		return th.Run(ctx, sc)
	})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	select {
	case <-ctx.Done():
	case <-interrupt:
		log.Println("signal: interrupt")

		cancel()
	}

	t := time.AfterFunc(2*time.Second, func() {
		os.Exit(1)
	})
	defer t.Stop()

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
