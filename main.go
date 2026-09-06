package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(model{message: "Hello, TUI!"})
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}