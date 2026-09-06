package main

import "sync"

// AppState holds things the background discovery loop produces and
// the TUI reads -- specifically, the running event log, which nothing
// else currently tracks.
type AppState struct {
	mu     sync.Mutex
	events []Event
}

func NewAppState() *AppState {
	return &AppState{}
}

func (s *AppState) AddEvent(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *AppState) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}