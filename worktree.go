package wtdetach

import (
	"bufio"
	"strings"
)

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	Branch string
}

// ParseWorktreeList parses the output of `git worktree list --porcelain`
func ParseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "worktree ") {
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			if current != nil {
				current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
			}
		} else if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
		}
	}

	// Handle the last worktree if there's no trailing newline
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

// FindWorktreeByBranch finds a worktree that has the specified branch checked out
// It excludes the worktree at excludePath
func FindWorktreeByBranch(worktrees []Worktree, branch, excludePath string) *Worktree {
	for _, wt := range worktrees {
		if wt.Branch == branch && wt.Path != excludePath {
			return &wt
		}
	}
	return nil
}
