package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// model holds everything the screen currently needs to know.
type model struct {
	message string
}

// Init runs once, when the program first starts.
func (m model) Init() tea.Cmd {
	return nil
}

// Update runs every time something happens (a keypress, etc.), and
// returns the new model plus an optional command to run next.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		m.message = "You pressed: " + msg.String()
	}
	return m, nil
}

// View turns the current model into the text actually drawn on screen.
func (m model) View() string {
	return m.message + "\n\n(press q to quit)"
}