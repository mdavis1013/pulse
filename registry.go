package main

import (
	"sort"
	"sync"
	"time"
)

// DeviceRecord is what we currently believe about one device.
type DeviceRecord struct {
	Instance    string
	ServiceType string
	IP          string
	Port        uint16
	LastSeen    time.Time
	TTL         time.Duration

	PTRSeenAt time.Time

	SRVSeenAt   time.Time
	SRVTarget   string
	SRVPriority uint16
	SRVWeight   uint16
	SRVTTL      time.Duration

	ASeenAt time.Time
	ATTL    time.Duration
}

// Registry tracks every currently-believed-live device.
type Registry struct {
	mu      sync.Mutex
	devices map[string]*DeviceRecord
}

func NewRegistry() *Registry {
	return &Registry{
		devices: make(map[string]*DeviceRecord),
	}
}

// Event describes a device joining or leaving.
type Event struct {
	Kind     string // "joined" or "left"
	Instance string
	At       time.Time
}

// Observe records that we just heard from this device, with the TTL
// that specific message declared. Returns a "joined" event if this is
// the first time we've seen it.
func (r *Registry) Observe(instance string, serviceType string, ttlSeconds uint32) *Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	ttl := time.Duration(ttlSeconds) * time.Second

	d, exists := r.devices[instance]
	var event *Event
	if !exists {
		d = &DeviceRecord{Instance: instance, ServiceType: serviceType}
		r.devices[instance] = d
		event = &Event{Kind: "joined", Instance: instance, At: now}
	}

	d.LastSeen = now
	d.TTL = ttl
	d.PTRSeenAt = now
	return event
}

// UpdateAddress fills in a device's resolved IP and port, once we've
// successfully cross-referenced its SRV and A records.
func (r *Registry) UpdateAddress(instance string, ip string, port uint16) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if d, ok := r.devices[instance]; ok {
		d.IP = ip
		d.Port = port
	}
}

// UpdateSRVDetail records the actual SRV record fields for a device.
func (r *Registry) UpdateSRVDetail(instance string, srv SRVData, ttlSeconds uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if d, ok := r.devices[instance]; ok {
		d.SRVSeenAt = time.Now()
		d.SRVTarget = srv.Target
		d.SRVPriority = srv.Priority
		d.SRVWeight = srv.Weight
		d.SRVTTL = time.Duration(ttlSeconds) * time.Second
	}
}

// UpdateADetail records the A record's own TTL for a device.
func (r *Registry) UpdateADetail(instance string, ttlSeconds uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if d, ok := r.devices[instance]; ok {
		d.ASeenAt = time.Now()
		d.ATTL = time.Duration(ttlSeconds) * time.Second
	}
}

// Sweep checks every known device against its own TTL, and evicts
// (with a "left" event) anything that's gone silent past it.
func (r *Registry) Sweep() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var events []Event

	for instance, d := range r.devices {
		if now.Sub(d.LastSeen) > d.TTL {
			delete(r.devices, instance)
			events = append(events, Event{Kind: "left", Instance: instance, At: now})
		}
	}

	return events
}

// Snapshot returns every currently-live device, for display purposes.
func (r *Registry) Snapshot() []DeviceRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]DeviceRecord, 0, len(r.devices))
	for _, d := range r.devices {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Instance < out[j].Instance
	})
	return out
}