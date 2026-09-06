package main

import (
	"fmt"
	"net"
	"time"
)

const mdnsAddr = "224.0.0.251:5353"

// Scanner can send mDNS queries and collect real replies from the network.
type Scanner struct {
	conn *net.UDPConn
}

// NewScanner opens a connection joined to the mDNS multicast group, so
// we can actually receive replies other devices send to that address.
func NewScanner() (*Scanner, error) {
	addr, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("joining mDNS multicast group (try running with sudo): %w", err)
	}

	return &Scanner{conn: conn}, nil
}

// Query sends one "who's out there?" packet, then collects real
// replies for the given amount of time.
func (s *Scanner) Query(name string, window time.Duration) ([]*Message, error) {
	dst, _ := net.ResolveUDPAddr("udp4", mdnsAddr)

	packet := EncodeQuery(name)
	_, err := s.conn.WriteToUDP(packet, dst)
	if err != nil {
		return nil, err
	}

	var messages []*Message
	deadline := time.Now().Add(window)
	buf := make([]byte, 65535)

	for time.Now().Before(deadline) {
		s.conn.SetReadDeadline(deadline)
		n, _, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			break
		}

		packetCopy := make([]byte, n)
		copy(packetCopy, buf[:n])

		msg, err := ParseMessage(packetCopy)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// Listen runs forever (until stop is closed), decoding every packet
// received and sending each one to the out channel. Meant to run in
// its own goroutine, so it can listen continuously while the rest of
// the program does other things at the same time.
func (s *Scanner) Listen(out chan<- *Message, stop <-chan struct{}) {
	buf := make([]byte, 65535)

	for {
		select {
		case <-stop:
			return
		default:
		}

		s.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		packetCopy := make([]byte, n)
		copy(packetCopy, buf[:n])

		msg, err := ParseMessage(packetCopy)
		if err != nil {
			continue
		}

		out <- msg
	}
}

// SendQuery fires a query without waiting for or collecting any
// replies -- replies are picked up separately by Listen, running
// concurrently.
func (s *Scanner) SendQuery(name string) error {
	dst, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return err
	}
	packet := EncodeQuery(name)
	_, err = s.conn.WriteToUDP(packet, dst)
	return err
}

