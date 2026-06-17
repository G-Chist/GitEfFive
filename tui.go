package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const logo = ` __    ___  ___  ___  ___         ___ 
/ _` + "`" + ` |  |  |__  |__  |__  | \  / |__  
\__> |  |  |___ |    |    |  \/  |___ `

type model struct {
	status     string
	running    bool
	saves      []SaveEntry
	saveCursor int
	saveOffset int
	width      int
	height     int
}

func initialModel() model {
	return model{status: "Ready."}
}

func (m model) Init() tea.Cmd {
	return doListSaves()
}

func (m model) View() string {
	var left strings.Builder

	for _, line := range strings.Split(logo, "\n") {
		left.WriteString("  ")
		left.WriteString(line)
		left.WriteString("\n")
	}
	left.WriteString("\n")
	left.WriteString("  ╔══════════════════════════════╗\n")
	left.WriteString("  ║  GitEfFive - Save Manager    ║\n")
	left.WriteString("  ╚══════════════════════════════╝\n")
	left.WriteString("\n")
	left.WriteString("  [s] quicksave    [q] quit\n")
	left.WriteString("\n")
	left.WriteString("  > ")
	left.WriteString(m.status)
	left.WriteString("\n")

	if m.width < 80 {
		return left.String()
	}

	rightW := m.width * 35 / 100
	if rightW < 36 {
		rightW = 36
	}
	if rightW > 56 {
		rightW = 56
	}
	contentW := rightW - 4

	var rb strings.Builder
	rb.WriteString("Saves\n")
	rb.WriteString(strings.Repeat("─", contentW) + "\n")

	if len(m.saves) == 0 {
		rb.WriteString("No saves yet\n")
	} else {
		maxVisible := m.height - 5
		if maxVisible < 1 {
			maxVisible = 1
		}

		end := m.saveOffset + maxVisible
		if end > len(m.saves) {
			end = len(m.saves)
		}

		for i := m.saveOffset; i < end; i++ {
			s := m.saves[i]
			msg := shortenMessage(s.Message)
			line := fmt.Sprintf("%s  %s", s.Hash, msg)
			maxLine := contentW - 2
			if len(line) > maxLine {
				line = line[:maxLine]
			}
			if i == m.saveCursor {
				rb.WriteString("▸ ")
			} else {
				rb.WriteString("  ")
			}
			rb.WriteString(line)
			rb.WriteString("\n")
		}
	}

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(rightW).
		Render(rb.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, left.String(), rightPanel)
}

func shortenMessage(msg string) string {
	if strings.HasPrefix(msg, "{SAVE ") && len(msg) >= 24 {
		return msg[:17] + "}"
	}
	if len(msg) > 20 {
		return msg[:17] + "..."
	}
	return msg
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.saveCursor > 0 {
				m.saveCursor--
				m.adjustSaveOffset()
			}
			return m, nil
		case "down", "j":
			if m.saveCursor < len(m.saves)-1 {
				m.saveCursor++
				m.adjustSaveOffset()
			}
			return m, nil
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
		return m, doListSaves()

	case listSavesResult:
		m.saves = msg.saves
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		return m, nil
	}

	return m, nil
}

func (m *model) adjustSaveOffset() {
	maxVisible := m.height - 5
	if maxVisible < 1 {
		maxVisible = 1
	}
	if m.saveCursor < m.saveOffset {
		m.saveOffset = m.saveCursor
	}
	if m.saveCursor >= m.saveOffset+maxVisible {
		m.saveOffset = m.saveCursor - maxVisible + 1
	}
}
