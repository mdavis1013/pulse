package main

import (
	"fmt"
	"time"
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
	devices  []DeviceRecord
	detail   bool

	selectedInstance string
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
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			if !m.detail {
				if d := m.selectedDevice(); d != nil {
					m.selectedInstance = d.Instance
				}
			}
			m.detail = !m.detail
			return m, nil
		case "esc":
			m.detail = false
			return m, nil
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

		case refreshMsg:
			devices := m.registry.Snapshot()
			m.devices = devices

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

	view := header + "\n\n" + body
	if m.detail {
		if d := m.deviceByInstance(m.selectedInstance); d != nil {
			view += "\n\n" + renderDetail(*d)
		}
	}
	return view
}

func (m model) selectedDevice() *DeviceRecord {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return nil
	}
	for i := range m.devices {
		if m.devices[i].Instance == row[0] {
			return &m.devices[i]
		}
	}
	return nil
}

func (m model) deviceByInstance(instance string) *DeviceRecord {
	for i := range m.devices {
		if m.devices[i].Instance == instance {
			return &m.devices[i]
		}
	}
	return nil
}

func renderDetail(d DeviceRecord) string {
	now := time.Now()

	content := fmt.Sprintf(
		"DEVICE DETAIL — %s\n\n"+
			"PTR RECORD (announces the instance exists)\n"+
			"  Service:  %s\n"+
			"  TTL:      %s\n\n"+
			"SRV RECORD (which host + port provides it)\n"+
			"  Target:   %s\n"+
			"  Port:     %d\n"+
			"  TTL:      %s\n\n"+
			"A RECORD (that host's actual IP)\n"+
			"  IP:       %s\n"+
			"  TTL:      %s",
		d.Instance,
		d.ServiceType, ttlRemaining(d.TTL, d.PTRSeenAt, now),
		d.SRVTarget, d.Port, ttlRemaining(d.SRVTTL, d.SRVSeenAt, now),
		d.IP, ttlRemaining(d.ATTL, d.ASeenAt, now),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(1, 2).
		Render(content)
}

// ttlRemaining computes how much longer a record should be trusted,
// based on real elapsed time since it was last seen -- the same core
// idea as the Registry's own failure detection, shown per-record here.
func ttlRemaining(ttl time.Duration, seenAt time.Time, now time.Time) string {
	if seenAt.IsZero() {
		return "(not seen yet)"
	}
	remaining := ttl - now.Sub(seenAt)
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("%ds left (of %ds)", int(remaining.Seconds()), int(ttl.Seconds()))
}