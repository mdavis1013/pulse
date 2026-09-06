package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	srvByInstance := map[string]SRVData{}
	ipByHost := map[string]string{}
	changed := make(chan struct{}, 1)

	messages := make(chan *Message, 100)
	stop := make(chan struct{})
	go scanner.Listen(messages, stop)

	scanner.SendQuery(metaService)

	// This goroutine is our existing discovery loop from before --
	// completely unchanged in what it DOES, except now, instead of
	// fmt.Println, it "rings the doorbell" by sending on changed.
	go func() {
		sweepTicker := time.NewTicker(5 * time.Second)
		defer sweepTicker.Stop()

		for {
			select {
			case msg := <-messages:
				for _, rr := range msg.Answers {
					switch rr.Type {
					case TypePTR:
						target, _, err := decodeName(msg.Raw, rr.rdataOffset)
						if err != nil {
							continue
						}

						if rr.Name == metaService {
							if !knownServiceTypes[target] {
								knownServiceTypes[target] = true
								scanner.SendQuery(target)
							}
							continue
						}

						registry.Observe(target, rr.Name, rr.TTL)
						ringDoorbell(changed)

					case TypeSRV:
						srv, err := rr.DecodeSRV(msg.Raw)
						if err != nil {
							continue
						}
						srvByInstance[rr.Name] = srv
						if ip, ok := ipByHost[srv.Target]; ok {
							registry.UpdateAddress(rr.Name, ip, srv.Port)
							ringDoorbell(changed)
						}

					case TypeA:
						ip, err := rr.DecodeA()
						if err != nil {
							continue
						}
						ipByHost[rr.Name] = ip
						for instance, srv := range srvByInstance {
							if srv.Target == rr.Name {
								registry.UpdateAddress(instance, ip, srv.Port)
								ringDoorbell(changed)
							}
						}
					}
				}
	
			case <-sweepTicker.C:
				events := registry.Sweep()
				if len(events) > 0 {
					ringDoorbell(changed)
				}
			}
		}
	}()

	p := tea.NewProgram(newModel(registry, changed))
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

// ringDoorbell sends a non-blocking signal -- if the person by the
// door is already mid-way through answering a previous ring, we don't
// want to sit here waiting for them to be ready; we just skip this
// ring, since a moment later they'll go re-check the registry anyway
// and see the latest state regardless.
func ringDoorbell(changed chan struct{}) {
	select {
	case changed <- struct{}{}:
	default:
	}
}