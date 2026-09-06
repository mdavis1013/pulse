package main

import (
	"fmt"
	"time"
)

const metaService = "_services._dns-sd._udp.local"

func main() {
	scanner, err := NewScanner()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	registry := NewRegistry()
	knownServiceTypes := map[string]bool{}

	messages := make(chan *Message, 100)
	stop := make(chan struct{})
	go scanner.Listen(messages, stop)

	scanner.SendQuery(metaService)

	sweepTicker := time.NewTicker(5 * time.Second)
	defer sweepTicker.Stop()

	fmt.Println("Listening continuously — Ctrl+C to stop")

	for {
		select {
		case msg := <-messages:
			for _, rr := range msg.Answers {
				if rr.Type != TypePTR {
					continue
				}
				target, _, err := decodeName(msg.Raw, rr.rdataOffset)
				if err != nil {
					continue
				}

				if rr.Name == metaService {
					// This tells us a whole SERVICE TYPE exists
					// (e.g. "_googlecast._tcp.local") -- not a device
					// yet. If it's new to us, immediately ask about
					// it specifically.
					if !knownServiceTypes[target] {
						knownServiceTypes[target] = true
						scanner.SendQuery(target)
					}
					continue
				}

				// Otherwise, this PTR answers a specific service type
				// we already asked about, and its target IS a real
				// device instance.
				if event := registry.Observe(target, rr.TTL); event != nil {
					fmt.Println("[+ joined]", event.Instance)
				}
			}

		case <-sweepTicker.C:
			for _, event := range registry.Sweep() {
				fmt.Println("[- left]", event.Instance)
			}
		}
	}
}