package main

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type quicksaveResult struct {
	output string
	err    error
}

type SaveEntry struct {
	Hash    string
	Date    string
	Message string
}

type listSavesResult struct {
	saves []SaveEntry
	err   error
}

func doQuicksave() tea.Cmd {
	return func() tea.Msg {
		now := time.Now().Format("2006-01-02-15-04-05")
		msg := "{SAVE " + now + "}"

		add := exec.Command("git", "add", ".")
		out1, err := add.CombinedOutput()
		if err != nil {
			return quicksaveResult{output: string(out1), err: err}
		}

		commit := exec.Command("git", "commit", "-m", msg)
		out2, err := commit.CombinedOutput()
		if err != nil {
			return quicksaveResult{output: string(out2), err: err}
		}

		push := exec.Command("git", "push", "origin", "main")
		out3, err := push.CombinedOutput()
		if err != nil {
			return quicksaveResult{output: string(out3), err: err}
		}

		return quicksaveResult{
			output: string(out1) + string(out2) + string(out3),
		}
	}
}

func doListSaves() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "log", "--grep={SAVE", "--format=%h|%as|%s", "--max-count=50")
		out, err := cmd.Output()
		if err != nil {
			return listSavesResult{err: err}
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			return listSavesResult{saves: []SaveEntry{}}
		}
		lines := strings.Split(s, "\n")
		saves := make([]SaveEntry, 0, len(lines))
		for _, line := range lines {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				saves = append(saves, SaveEntry{
					Hash:    parts[0],
					Date:    parts[1],
					Message: parts[2],
				})
			}
		}
		return listSavesResult{saves: saves}
	}
}
