package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// minimum terminal width before the layout collapses to single-column
	minWidth        = 40
	// threshold below which only the logo+status+footer (no panels) are shown
	narrowThreshold = 80

	// saves panel: width = terminal × pct / 100, clamped to [min, max]
	savesPanelPct   = 28
	savesPanelMin   = 28
	savesPanelMax   = 48
	// players panel: width = terminal × pct / 100, clamped to [min, max]
	playersPanelPct = 22
	playersPanelMin = 20
	playersPanelMax = 36
	// branches panel: width = terminal × pct / 100, clamped to [min, max]
	branchesPanelPct = 28
	branchesPanelMin = 24
	branchesPanelMax = 40

	borderPadding  = 4  // total horizontal padding consumed by the lipgloss border (2 per side × 2)
	reservedLines  = 5  // lines reserved for header+footer in each panel's visible area
	minVisible     = 1  // at least one row must be visible regardless of terminal height
	dateWidth      = 10 // printed width of the YYYY-MM-DD date column in the saves panel
	blocksWidth    = 6  // width of the visual progress-bar column in both panels
	countWidth     = 2  // printed width of the numeric counter in the players panel
	cursorWidth    = 2  // width of the cursor indicator ("▸ " or "  ") prepended to each row
	ellipsisLen    = 3  // length of the "..." truncation marker

	// messages starting with "{SAVE " and at least this long get special truncation
	saveMsgTrunc = 17 // prefix length kept for SAVE messages (including the trailing "}")
	saveMsgLen   = 24 // minimum total length to qualify as a SAVE commit message
	maxMsgLen    = 20 // generic messages longer than this are truncated to saveMsgTrunc + "..."

	focusedSaves   = 0 // focused value for the saves panel
	focusedPlayers = 1 // focused value for the players panel
	focusedBranches = 2 // focused value for the branches panel
)

const logo = ` __    ___  ___  ___  ___         ___ 
/ _` + "`" + ` |  |  |__  |__  |__  | \  / |__  
\__> |  |  |___ |    |    |  \/  |___ `

// model implements the tea.Model interface. Bubble Tea requires three methods:
//
//   Init() tea.Cmd                         called once at startup; may return a command
//   Update(tea.Msg) (tea.Model, tea.Cmd)   handle messages, mutate state
//   View() string            		    render the current state to a terminal string
//
type model struct {
	status         string
	running        bool
	saves          []SaveEntry
	saveCursor     int
	saveOffset     int
	players          []PlayerEntry
	playersCursor    int
	playersOffset    int
	branches         []BranchEntry
	branchesCursor   int
	branchesOffset   int
	repoStatus       RepoStatus
	focused          int
	width          int
	height         int
}

// initialModel returns the starting state before any messages are processed.
func initialModel() model {
	return model{status: "Ready."}
}

// Init is called once when the Bubble Tea program starts. Returning
// doListSaves() here means the commit list is loaded immediately on launch.
// tea.Cmd is just "func() tea.Msg", a thunk that does work and sends the
// result back as a message. If you don't need to do anything on startup,
// return nil.
func (m model) Init() tea.Cmd {
	return tea.Batch(doListSaves(), doListPlayers(), doListBranches(), doRepoStatus())
}

func (m model) View() string {
	w := m.width
	if w < minWidth {
		w = minWidth
	}

	// Logo banner (centered horizontally)
	var logoBanner strings.Builder
	for _, line := range strings.Split(logo, "\n") {
		logoBanner.WriteString(
			lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(line),
		)
		logoBanner.WriteString("\n")
	}

	// Status line
	statusLine := fmt.Sprintf("  %s", m.status)

	// Footer with hotkeys (displayed at the bottom)
	footer := lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Render("[s] quicksave [l] load [r] quickload [q] quit")

	if m.width < narrowThreshold {
		logoBanner.WriteString("\n")
		return logoBanner.String() + statusLine + "\n\n" + footer + "\n"
	}

	// Saves panel
	savesW := m.width * savesPanelPct / 100
	if savesW < savesPanelMin {
		savesW = savesPanelMin
	}
	if savesW > savesPanelMax {
		savesW = savesPanelMax
	}
	contentW := savesW - borderPadding

	var rb strings.Builder
	rb.WriteString("Saves\n")
	rb.WriteString(strings.Repeat("─", contentW) + "\n")

	if len(m.saves) == 0 {
		rb.WriteString("No saves yet\n")
	} else {
		maxVisible := m.height - reservedLines
		if maxVisible < minVisible {
			maxVisible = minVisible
		}

		end := m.saveOffset + maxVisible
		if end > len(m.saves) {
			end = len(m.saves)
		}

		msgW := contentW - cursorWidth - dateWidth - 2 - 1 - blocksWidth

		for i := m.saveOffset; i < end; i++ {
			s := m.saves[i]
			msg := shortenMessage(s.Message)
			blocks := strings.Repeat("█", s.Blocks) + strings.Repeat("░", blocksWidth-s.Blocks)
			date := fmt.Sprintf("%-*s", dateWidth, s.Date)

			if len(msg) > msgW {
				if msgW > ellipsisLen {
					msg = msg[:msgW-ellipsisLen] + "..."
				} else {
					msg = msg[:msgW]
				}
			}
			msg = fmt.Sprintf("%-*s", msgW, msg)

			line := fmt.Sprintf("%s  %s %s", date, msg, blocks)
			if i == m.saveCursor {
				rb.WriteString("▸ ")
			} else {
				rb.WriteString("  ")
			}
			rb.WriteString(line)
			rb.WriteString("\n")
		}
	}

	savesPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(savesW).
		Render(rb.String())

	// Players panel
	playersW := m.width * playersPanelPct / 100
	if playersW < playersPanelMin {
		playersW = playersPanelMin
	}
	if playersW > playersPanelMax {
		playersW = playersPanelMax
	}
	playersContentW := playersW - borderPadding

	var pb strings.Builder
	pb.WriteString("Players\n")
	pb.WriteString(strings.Repeat("─", playersContentW) + "\n")

	if len(m.players) == 0 {
		pb.WriteString("No players\n")
	} else {
		maxVisible := m.height - reservedLines
		if maxVisible < minVisible {
			maxVisible = minVisible
		}

		end := m.playersOffset + maxVisible
		if end > len(m.players) {
			end = len(m.players)
		}

		nameW := playersContentW - cursorWidth - 1 - countWidth - 1 - blocksWidth

		for i := m.playersOffset; i < end; i++ {
			p := m.players[i]
			blocks := strings.Repeat("█", p.Blocks) + strings.Repeat("░", blocksWidth-p.Blocks)
			name := p.Name
			if len(name) > nameW {
				name = name[:nameW]
			}
			name = fmt.Sprintf("%-*s", nameW, name)
			count := fmt.Sprintf("%*d", countWidth, p.Count)
			if i == m.playersCursor {
				pb.WriteString("▸ ")
			} else {
				pb.WriteString("  ")
			}
			pb.WriteString(fmt.Sprintf("%s %s %s", name, count, blocks))
			pb.WriteString("\n")
		}
	}

	playersPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(playersW).
		Render(pb.String())

	// Branches panel
	branchesW := m.width * branchesPanelPct / 100
	if branchesW < branchesPanelMin {
		branchesW = branchesPanelMin
	}
	if branchesW > branchesPanelMax {
		branchesW = branchesPanelMax
	}
	branchesContentW := branchesW - borderPadding

	var bb strings.Builder
	bb.WriteString("Branches\n")
	bb.WriteString(strings.Repeat("─", branchesContentW) + "\n")

	if len(m.branches) == 0 {
		bb.WriteString("No branches\n")
	} else {
		maxVisible := m.height - reservedLines
		if maxVisible < minVisible {
			maxVisible = minVisible
		}

		end := m.branchesOffset + maxVisible
		if end > len(m.branches) {
			end = len(m.branches)
		}

		nameW := branchesContentW - cursorWidth - 2

		for i := m.branchesOffset; i < end; i++ {
			b := m.branches[i]
			name := b.Name
			if len(name) > nameW {
				name = name[:nameW]
			}
			name = fmt.Sprintf("%-*s", nameW, name)

			marker := "  "
			if b.Current {
				marker = " *"
			}

			if i == m.branchesCursor {
				bb.WriteString("▸ ")
			} else {
				bb.WriteString("  ")
			}
			bb.WriteString(fmt.Sprintf("%s%s", name, marker))
			bb.WriteString("\n")
		}
	}

	branchesPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(branchesW).
		Render(bb.String())

	body := lipgloss.JoinHorizontal(lipgloss.Top, savesPanel, playersPanel, branchesPanel)

	// Git status bar
	st := m.repoStatus
	var statusBar string
	if st.Branch != "" {
		var parts []string
		parts = append(parts, fmt.Sprintf("branch: %s", st.Branch))
		if st.Modified > 0 || st.Added > 0 || st.Deleted > 0 || st.Untracked > 0 {
			var fileParts []string
			if st.Modified > 0 {
				fileParts = append(fileParts, fmt.Sprintf("%d modified", st.Modified))
			}
			if st.Added > 0 {
				fileParts = append(fileParts, fmt.Sprintf("%d added", st.Added))
			}
			if st.Deleted > 0 {
				fileParts = append(fileParts, fmt.Sprintf("%d deleted", st.Deleted))
			}
			if st.Untracked > 0 {
				fileParts = append(fileParts, fmt.Sprintf("%d untracked", st.Untracked))
			}
			parts = append(parts, "files: "+strings.Join(fileParts, ", "))
		}
		if st.LinesAdded > 0 || st.LinesDeleted > 0 {
			parts = append(parts, fmt.Sprintf("lines: +%d -%d", st.LinesAdded, st.LinesDeleted))
		}
		statusBar = " " + strings.Join(parts, "\n ")
	}

	return lipgloss.JoinVertical(lipgloss.Top, logoBanner.String(), "", statusLine, "", body, "", statusBar, "", footer)
}

// shortenMessage truncates commit messages for display. SAVE commits
// (which have a long timestamp) are trimmed to the first 17 chars + "}".
// Other long messages get "..." ellipsis at 20 chars.
func shortenMessage(msg string) string {
	if strings.HasPrefix(msg, "{SAVE ") && len(msg) >= saveMsgLen {
		return msg[:saveMsgTrunc] + "}"
	}
	if len(msg) > maxMsgLen {
		return msg[:saveMsgTrunc] + "..."
	}
	return msg
}

// Update is called every time a message arrives. This is the only place where
// state is mutated. It returns the new model (or the same one) and optionally
// a tea.Cmd to perform side effects (git operations, etc.).
//
// Type-switching on msg allows handling different message types:
//   tea.WindowSizeMsg is sent when the terminal is resized
//   tea.KeyMsg        is sent on key presses
//   quicksaveResult   is our custom message from doQuicksave()
//   listSavesResult   is our custom message from doListSaves()
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Bubble Tea sends this automatically on resize and on startup
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			switch m.focused {
			case focusedSaves:
				if m.saveCursor > 0 {
					m.saveCursor--
					m.adjustSaveOffset()
				}
			case focusedPlayers:
				if m.playersCursor > 0 {
					m.playersCursor--
					m.adjustPlayersOffset()
				}
			case focusedBranches:
				if m.branchesCursor > 0 {
					m.branchesCursor--
					m.adjustBranchesOffset()
				}
			}
			return m, nil
		case "down", "j":
			switch m.focused {
			case focusedSaves:
				if m.saveCursor < len(m.saves)-1 {
					m.saveCursor++
					m.adjustSaveOffset()
				}
			case focusedPlayers:
				if m.playersCursor < len(m.players)-1 {
					m.playersCursor++
					m.adjustPlayersOffset()
				}
			case focusedBranches:
				if m.branchesCursor < len(m.branches)-1 {
					m.branchesCursor++
					m.adjustBranchesOffset()
				}
			}
			return m, nil
		case "tab":
			m.focused = (m.focused + 1) % 3
			return m, nil
		case "left":
			if m.focused > 0 {
				m.focused--
			}
			return m, nil
		case "right":
			if m.focused < 2 {
				m.focused++
			}
			return m, nil
		case "r":
			if !m.running {
				m.running = true
				m.status = "Running quickload..."
				return m, doQuickload()
			}
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
		// After a save, refresh panels and status.
		return m, tea.Batch(doListSaves(), doListBranches(), doRepoStatus())

	case quickloadResult:
		m.running = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", strings.TrimSpace(msg.err.Error()))
		} else {
			m.status = "Quickload complete!"
		}
		return m, nil

	case listSavesResult:
		m.saves = msg.saves
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		return m, nil

	case listPlayersResult:
		m.players = msg.players
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		return m, nil

	case listBranchesResult:
		m.branches = msg.branches
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
		}
		// set cursor on the current branch
		for i, b := range m.branches {
			if b.Current {
				m.branchesCursor = i
				break
			}
		}
		return m, nil

	case repoStatusResult:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
		} else {
			m.repoStatus = msg.status
		}
		return m, nil
	}

	return m, nil
}

// adjustSaveOffset implements virtual scrolling for the saves list.
// It ensures the cursor is always visible within the viewport by adjusting
// the saveOffset (the index of the first visible item) when the cursor moves
// above or below the visible window.
//
// This is called on every cursor movement (up/down in Update).
// It takes a pointer receiver (*model) because it mutates m.saveOffset.
func (m *model) adjustSaveOffset() {
	maxVisible := m.height - reservedLines
	if maxVisible < minVisible {
		maxVisible = minVisible
	}
	if m.saveCursor < m.saveOffset {
		m.saveOffset = m.saveCursor
	}
	if m.saveCursor >= m.saveOffset+maxVisible {
		m.saveOffset = m.saveCursor - maxVisible + 1
	}
}

func (m *model) adjustPlayersOffset() {
	maxVisible := m.height - reservedLines
	if maxVisible < minVisible {
		maxVisible = minVisible
	}
	if m.playersCursor < m.playersOffset {
		m.playersOffset = m.playersCursor
	}
	if m.playersCursor >= m.playersOffset+maxVisible {
		m.playersOffset = m.playersCursor - maxVisible + 1
	}
}

func (m *model) adjustBranchesOffset() {
	maxVisible := m.height - reservedLines
	if maxVisible < minVisible {
		maxVisible = minVisible
	}
	if m.branchesCursor < m.branchesOffset {
		m.branchesOffset = m.branchesCursor
	}
	if m.branchesCursor >= m.branchesOffset+maxVisible {
		m.branchesOffset = m.branchesCursor - maxVisible + 1
	}
}
