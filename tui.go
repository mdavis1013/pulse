package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// refreshMsg is sent whenever new data is available for the model to
// pick up -- our own custom message type, since bubbletea only knows
// about the built-in ones (keypresses, etc.) unless we define more.
type refreshMsg struct{}

// model holds everything the screen currently needs to know.
type model struct {
	registry *Registry
	changed  <-chan struct{}
	devices  []DeviceRecord
}

// waitForChange blocks until something arrives on the changed channel,
// then reports it to bubbletea as a refreshMsg. bubbletea re-runs this
// itself after every Update, so this effectively keeps listening
// forever, one wakeup at a time.
func waitForChange(changed <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-changed
		return refreshMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return waitForChange(m.changed)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}

	case refreshMsg:
		m.devices = m.registry.Snapshot()
		return m, waitForChange(m.changed) // keep listening for the NEXT change
	}
	return m, nil
}

func (m model) View() string {
	out := "pulse -- live device discovery (q to quit)\n\n"
	for _, d := range m.devices {
		out += d.Instance + "\n"
	}
	return out
}