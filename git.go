package main

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type quicksaveResult struct {
	output string
	err    error
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

		commit := exec.Command("git", "commit", "-am", msg)
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
