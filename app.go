package main

import tea "github.com/charmbracelet/bubbletea"

func main() {
	// tea.NewProgram creates a Bubble Tea TUI application.
	// It implements the Elm Architecture: Model → Update → View loop.
	// initialModel() returns the starting state (tea.Model interface).
	// tea.WithAltScreen() switches to the terminal's alternate screen buffer,
	// so the TUI replaces the terminal content temporarily instead of scrolling.
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	// p.Run() starts the event loop. It blocks until tea.Quit is sent.
	// The returned tea.Model holds the final state after the program exits.
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

