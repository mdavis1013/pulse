package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// exportDevice mirrors DeviceRecord, but with durations written as
// readable strings ("1h15m0s") instead of raw nanosecond numbers --
// this file is meant to be something a human could actually open and
// read, not just technically-correct data.
type exportDevice struct {
	Instance    string `json:"instance"`
	ServiceType string `json:"service_type"`
	IP          string `json:"ip"`
	Port        uint16 `json:"port"`
	TTL         string `json:"ttl"`
	SRVTarget   string `json:"srv_target,omitempty"`
	SRVTTL      string `json:"srv_ttl,omitempty"`
	ATTL        string `json:"a_ttl,omitempty"`
}

type exportEvent struct {
	Kind     string `json:"kind"`
	Instance string `json:"instance"`
	At       string `json:"at"`
}

type ExportSnapshot struct {
	ExportedAt string         `json:"exported_at"`
	Devices    []exportDevice `json:"devices"`
	Events     []exportEvent  `json:"events"`
}

func exportSnapshot(registry *Registry, appState *AppState) (string, error) {
	devices := registry.Snapshot()
	events := appState.Events()

	snap := ExportSnapshot{ExportedAt: time.Now().Format(time.RFC3339)}

	for _, d := range devices {
		snap.Devices = append(snap.Devices, exportDevice{
			Instance:    d.Instance,
			ServiceType: d.ServiceType,
			IP:          d.IP,
			Port:        d.Port,
			TTL:         d.TTL.String(),
			SRVTarget:   d.SRVTarget,
			SRVTTL:      d.SRVTTL.String(),
			ATTL:        d.ATTL.String(),
		})
	}

	for _, e := range events {
		snap.Events = append(snap.Events, exportEvent{
			Kind:     e.Kind,
			Instance: e.Instance,
			At:       e.At.Format(time.RFC3339),
		})
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding snapshot: %w", err)
	}

	filename := fmt.Sprintf("pulse-export-%s.json", time.Now().Format("2006-01-02-150405"))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return filename, nil
}