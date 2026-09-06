package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ExportSnapshot is what gets written to disk -- a durable record of
// what was observed, readable outside the running program.
type ExportSnapshot struct {
	ExportedAt string         `json:"exported_at"`
	Devices    []DeviceRecord `json:"devices"`
	Events     []Event        `json:"events"`
}

// exportSnapshot writes the current state to a timestamped JSON file
// and returns its filename.
func exportSnapshot(registry *Registry, appState *AppState) (string, error) {
	snap := ExportSnapshot{
		ExportedAt: time.Now().Format(time.RFC3339),
		Devices:    registry.Snapshot(),
		Events:     appState.Events(),
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