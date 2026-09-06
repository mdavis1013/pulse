package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)
var (
	colorMuted = lipgloss.Color("#8a8fa3")
	colorBlue  = lipgloss.Color("#5b8def")
	colorText  = lipgloss.Color("#e6e6e6")
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
	header := lipgloss.NewStyle().Bold(true).Foreground(colorText).Render("pulse") +
		"  " +
		lipgloss.NewStyle().Foreground(colorMuted).Render("live device discovery — q to quit")

	tableBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Render(m.table.View())

	eventsTitle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted).Render("RECENT EVENTS")

	start := 0
	if len(m.events) > 10 {
		start = len(m.events) - 10
	}
	eventLines := ""
	for i := len(m.events) - 1; i >= start; i-- {
		e := m.events[i]
		color := colorBlue
		eventLines += lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("[%s] %s", e.Kind, e.Instance)) + "\n"
	}

	eventsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Width(40).
		Render(eventsTitle + "\n\n" + eventLines)

	body := lipgloss.JoinHorizontal(lipgloss.Top, tableBox, "  ", eventsBox)

	return header + "\n\n" + body
}