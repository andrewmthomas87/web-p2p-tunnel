package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/andrewmthomas87/web-p2p-tunnel/internal/signaling"
	"github.com/andrewmthomas87/web-p2p-tunnel/internal/tunnel"
)

var (
	signalingServerURLStr = flag.String("signaling-server-url", "http://localhost:8080", "signaling server url")
)

func main() {
	flag.Parse()

	signalingServerURL, err := url.Parse(*signalingServerURLStr)
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

	th := tunnel.NewHub()

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := sc.Run(done); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := th.Run(
			done,
			sc.Offers(),
			sc.Answers(),
			sc.RemoteICECandidates(),
			sc.LocalICECandidates(),
		); err != nil {
			log.Fatal(err)
		}
	}()

	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	select {
	case <-wgDone:
	case <-interrupt:
		log.Println("signal: interrupt")

		close(done)

		select {
		case <-wgDone:
		case <-time.After(2 * time.Second):
		}
	}
}
