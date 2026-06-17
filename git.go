package main

import (
	"os/exec"
	"strconv"
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
	Size    int
	Blocks  int
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
		cmd := exec.Command("git", "log", "--numstat", "--format=%h|%as|%s", "--max-count=50")
		out, err := cmd.Output()
		if err != nil {
			return listSavesResult{err: err}
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			return listSavesResult{saves: []SaveEntry{}}
		}

		lines := strings.Split(s, "\n")
		var saves []SaveEntry
		maxSize := 0

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 && looksLikeHash(parts[0]) {
				e := SaveEntry{
					Hash:    parts[0],
					Date:    parts[1],
					Message: parts[2],
				}

				for i+1 < len(lines) {
					next := strings.TrimSpace(lines[i+1])
					if next == "" {
						i++
						continue
					}
					fields := strings.Fields(next)
					if len(fields) >= 2 && isNumstatField(fields[0]) && isNumstatField(fields[1]) {
						if fields[0] != "-" {
							if n, err := strconv.Atoi(fields[0]); err == nil {
								e.Size += n
							}
						}
						if fields[1] != "-" {
							if n, err := strconv.Atoi(fields[1]); err == nil {
								e.Size += n
							}
						}
						i++
					} else {
						break
					}
				}

				if e.Size > maxSize {
					maxSize = e.Size
				}
				saves = append(saves, e)
			}
		}

		for i := range saves {
			if maxSize > 0 {
				saves[i].Blocks = 1 + saves[i].Size*5/maxSize
				if saves[i].Blocks > 6 {
					saves[i].Blocks = 6
				}
			} else {
				saves[i].Blocks = 1
			}
		}

		return listSavesResult{saves: saves}
	}
}

func looksLikeHash(s string) bool {
	if len(s) < 7 || len(s) > 12 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func isNumstatField(s string) bool {
	if s == "-" {
		return true
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
