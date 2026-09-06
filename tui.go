package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type refreshMsg struct{}

type model struct {
	registry *Registry
	appState *AppState
	changed  <-chan struct{}
	table    table.Model
	events   []Event
}

func newModel(registry *Registry, appState *AppState, changed <-chan struct{}) model {
	columns := []table.Column{
		{Title: "DEVICE", Width: 40},
		{Title: "SERVICE", Width: 25},
		{Title: "ADDRESS", Width: 22},
	}
	t := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithHeight(15))

	return model{registry: registry, appState: appState, changed: changed, table: t}
}

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
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case refreshMsg:
		devices := m.registry.Snapshot()
		rows := make([]table.Row, 0, len(devices))
		for _, d := range devices {
			addr := "(resolving...)"
			if d.IP != "" {
				addr = fmt.Sprintf("%s:%d", d.IP, d.Port)
			}
			rows = append(rows, table.Row{d.Instance, d.ServiceType, addr})
		}
		m.table.SetRows(rows)

		m.events = m.appState.Events()

		return m, waitForChange(m.changed)
	}
	return m, nil
}

func (m model) View() string {
	eventLines := "RECENT EVENTS\n\n"
	start := 0
	if len(m.events) > 10 {
		start = len(m.events) - 10
	}
	for i := len(m.events) - 1; i >= start; i-- {
		e := m.events[i]
		eventLines += fmt.Sprintf("[%s] %s\n", e.Kind, e.Instance)
	}

	return fmt.Sprintf("pulse -- live device discovery (q to quit)\n\n%s\n\n%s", m.table.View(), eventLines)
}