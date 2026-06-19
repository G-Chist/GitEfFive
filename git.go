package main

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	// "tea" is the standard alias for Bubble Tea, the TUI framework.
	// Key types:
	//   tea.Model  is an interface { Init() tea.Cmd; Update(tea.Msg) (tea.Model, tea.Cmd); View() string }
	//   tea.Cmd    is a function that does async work and returns a tea.Msg  ("command" in Elm terms)
	//   tea.Msg    is any value passed through Update to trigger state changes ("message" in Elm terms)
	tea "github.com/charmbracelet/bubbletea"
)

// quicksaveResult is a custom tea.Msg type. When doQuicksave() finishes
// running the git commands, it sends this message back to the Update loop.
// The Update method (in tui.go) matches on this type to update the UI state.
type quicksaveResult struct {
	output string
	err    error
}

type quickloadResult struct {
	output string
	err    error
}

// SaveEntry holds parsed data for one git commit loaded from git log.
// tea.Msg types can be any Go struct
type SaveEntry struct {
	Hash    string
	Date    string
	Message string
	Size    int // total lines added + deleted (from --numstat)
	Blocks  int // visual bar width proportional to Size
}

// listSavesResult is another tea.Msg type. doListSaves() sends this back
// after parsing git log output into a slice of SaveEntry.
type listSavesResult struct {
	saves []SaveEntry
	err   error
}

type PlayerEntry struct {
	Name   string
	Count  int
	Blocks int
}

type listPlayersResult struct {
	players []PlayerEntry
	err     error
}

type BranchEntry struct {
	Name    string
	Current bool
}

type listBranchesResult struct {
	branches []BranchEntry
	err      error
}

type RepoStatus struct {
	Branch      string
	Modified    int
	Added       int
	Deleted     int
	Untracked   int
	LinesAdded  int
	LinesDeleted int
}

type repoStatusResult struct {
	status RepoStatus
	err    error
}

func doRepoStatus() tea.Cmd {
	return func() tea.Msg {
		// get current branch
		branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		branchOut, err := branchCmd.Output()
		if err != nil {
			return repoStatusResult{err: err}
		}
		branch := strings.TrimSpace(string(branchOut))

		// count files by status
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusOut, err := statusCmd.Output()
		if err != nil {
			return repoStatusResult{err: err}
		}
		var modified, added, deleted, untracked int
		for _, line := range strings.Split(string(statusOut), "\n") {
			if len(line) < 2 {
				continue
			}
			idx := line[0]
			wt := line[1]
			if line[:2] == "??" {
				untracked++
			} else {
				if idx == 'M' || wt == 'M' {
					modified++
				}
				if idx == 'A' || wt == 'A' {
					added++
				}
				if idx == 'D' || wt == 'D' {
					deleted++
				}
			}
		}

		// count lines added/deleted in working tree
		diffCmd := exec.Command("git", "diff", "--numstat")
		diffOut, err := diffCmd.Output()
		var linesAdded, linesDeleted int
		if err == nil {
			for _, line := range strings.Split(string(diffOut), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if n, err := strconv.Atoi(fields[0]); err == nil {
						linesAdded += n
					}
					if n, err := strconv.Atoi(fields[1]); err == nil {
						linesDeleted += n
					}
				}
			}
		}

		return repoStatusResult{
			status: RepoStatus{
				Branch:       branch,
				Modified:     modified,
				Added:        added,
				Deleted:      deleted,
				Untracked:    untracked,
				LinesAdded:   linesAdded,
				LinesDeleted: linesDeleted,
			},
		}
	}
}

func doListBranches() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "branch", "--format=%(refname:short)|%(HEAD)")
		out, err := cmd.Output()
		if err != nil {
			return listBranchesResult{err: err}
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			return listBranchesResult{}
		}
		lines := strings.Split(s, "\n")
		branches := make([]BranchEntry, 0, len(lines))
		for _, line := range lines {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) != 2 {
				continue
			}
			branches = append(branches, BranchEntry{
				Name:    parts[0],
				Current: parts[1] == "*",
			})
		}
		return listBranchesResult{branches: branches}
	}
}

func doListPlayers() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "log", "--format=%an", "--max-count=50")
		out, err := cmd.Output()
		if err != nil {
			return listPlayersResult{err: err}
		}
		s := strings.TrimSpace(string(out))
		if s == "" {
			return listPlayersResult{}
		}
		lines := strings.Split(s, "\n")
		counts := make(map[string]int)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				counts[line]++
			}
		}
		var players []PlayerEntry
		maxCount := 0
		for name, c := range counts {
			if c > maxCount {
				maxCount = c
			}
			players = append(players, PlayerEntry{Name: name, Count: c})
		}
		sort.Slice(players, func(i, j int) bool {
			return players[i].Count > players[j].Count
		})
		for i := range players {
			if maxCount > 0 {
				players[i].Blocks = 1 + players[i].Count*5/maxCount
			} else {
				players[i].Blocks = 1
			}
		}
		return listPlayersResult{players: players}
	}
}

// doQuicksave returns a tea.Cmd, a closure that runs async (but not in a
// goroutine; Bubble Tea runs these synchronously on a separate goroutine for
// us). When called, it runs "git add .", "git commit", and "git push origin
// main" sequentially. The result is wrapped in a quicksaveResult message and
// sent back through the Update loop.
//
// tea.Cmd is simply: type Cmd func() Msg
// Any function that matches that signature can be returned from Update or Init
// to trigger side effects. The returned Msg is then fed back into Update.
func doQuicksave() tea.Cmd {
	return func() tea.Msg {
		// Go time formatting uses the reference time Mon Jan 2 15:04:05 MST 2006.
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

		branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		branchOut, err := branchCmd.Output()
		if err != nil {
			return quicksaveResult{err: err}
		}
		branch := strings.TrimSpace(string(branchOut))

		push := exec.Command("git", "push", "origin", branch)
		out3, err := push.CombinedOutput()
		if err != nil {
			return quicksaveResult{output: string(out3), err: err}
		}

		return quicksaveResult{
			output: string(out1) + string(out2) + string(out3),
		}
	}
}

func doQuickload() tea.Cmd {
	return func() tea.Msg {
		stash := exec.Command("git", "stash")
		out, err := stash.CombinedOutput()
		return quickloadResult{output: string(out), err: err}
	}
}

// doListSaves returns a tea.Cmd that runs "git log --numstat --max-count=50".
// The output format includes the abbreviated hash, date, and subject on one
// line, followed by lines of "added deleted filename" (the --numstat data)
// for each commit.
//
// The function parses this into []SaveEntry and wraps it in a listSavesResult
// message. The View method then renders the list.
func doListSaves() tea.Cmd {
	return func() tea.Msg {
		// --format=%h|%as|%s gives "abc1234|2026-06-17|commit message"
		// --numstat follows each commit header with "N\tN\tfilename" lines
		// --max-count=50 limits to the 50 most recent commits
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

			// Detect a commit header by splitting on "|" and checking if the
			// first field looks like a git hash (hex string 7-12 chars).
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 && looksLikeHash(parts[0]) {
				e := SaveEntry{
					Hash:    parts[0],
					Date:    parts[1],
					Message: parts[2],
				}

				// Consume subsequent --numstat lines (e.g. "3\t2\tmain.go")
				// that belong to this commit, summing added+deleted lines as Size.
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

		// Normalise Size to a Blocks value 1-6 for the visual bar in the UI.
		// The largest commit gets 6 blocks; others scale proportionally.
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

// looksLikeHash checks if s looks like an abbreviated git commit hash,
// i.e. a hex string between 7 and 12 characters. Used to distinguish
// commit-header lines from --numstat lines in the git log output.
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

// isNumstatField checks whether a string is either "-" or a non-negative
// integer. Git's --numstat uses "-" for binary files and digits for text
// files. This helps us distinguish numstat data lines from other output.
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
