package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const logo = ` __    ___  ___  ___  ___         ___ 
/ _` + "`" + ` |  |  |__  |__  |__  | \  / |__  
\__> |  |  |___ |    |    |  \/  |___ `

type model struct {
	status  string
	running bool
}

func initialModel() model {
	return model{status: "Ready."}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) View() string {
	var b strings.Builder

	for _, line := range strings.Split(logo, "\n") {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  ╔══════════════════════════════╗\n")
	b.WriteString("  ║  GitEfFive - Save Manager   ║\n")
	b.WriteString("  ╚══════════════════════════════╝\n")
	b.WriteString("\n")
	b.WriteString("  [s] quicksave    [q] quit\n")
	b.WriteString("\n")
	b.WriteString("  > ")
	b.WriteString(m.status)
	b.WriteString("\n")

	return b.String()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			if !m.running {
				m.running = true
				m.status = "Running quicksave..."
				return m, doQuicksave()
			}
		}
	case quicksaveResult:
		m.running = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", strings.TrimSpace(msg.err.Error()))
		} else {
			m.status = "Quicksave complete!"
		}
	}
	return m, nil
}
