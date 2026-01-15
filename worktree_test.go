package wtdetach

import (
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Worktree
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name: "single worktree",
			input: `worktree /path/to/repo
HEAD abc123
branch refs/heads/main

`,
			expected: []Worktree{
				{Path: "/path/to/repo", Branch: "main"},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /path/to/repo
HEAD abc123
branch refs/heads/main

worktree /path/to/worktree1
HEAD def456
branch refs/heads/feature-x

worktree /path/to/worktree2
HEAD 789ghi
branch refs/heads/feature-y

`,
			expected: []Worktree{
				{Path: "/path/to/repo", Branch: "main"},
				{Path: "/path/to/worktree1", Branch: "feature-x"},
				{Path: "/path/to/worktree2", Branch: "feature-y"},
			},
		},
		{
			name: "detached HEAD worktree",
			input: `worktree /path/to/repo
HEAD abc123
branch refs/heads/main

worktree /path/to/detached
HEAD def456
detached

`,
			expected: []Worktree{
				{Path: "/path/to/repo", Branch: "main"},
				{Path: "/path/to/detached", Branch: ""},
			},
		},
		{
			name: "no trailing newline",
			input: `worktree /path/to/repo
HEAD abc123
branch refs/heads/main`,
			expected: []Worktree{
				{Path: "/path/to/repo", Branch: "main"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWorktreeList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d worktrees, got %d", len(tt.expected), len(result))
				return
			}
			for i, wt := range result {
				if wt.Path != tt.expected[i].Path {
					t.Errorf("worktree[%d].Path: expected %q, got %q", i, tt.expected[i].Path, wt.Path)
				}
				if wt.Branch != tt.expected[i].Branch {
					t.Errorf("worktree[%d].Branch: expected %q, got %q", i, tt.expected[i].Branch, wt.Branch)
				}
			}
		})
	}
}

func TestFindWorktreeByBranch(t *testing.T) {
	worktrees := []Worktree{
		{Path: "/path/to/repo", Branch: "main"},
		{Path: "/path/to/worktree1", Branch: "feature-x"},
		{Path: "/path/to/worktree2", Branch: "feature-y"},
	}

	tests := []struct {
		name        string
		branch      string
		excludePath string
		expected    *Worktree
	}{
		{
			name:        "find existing branch",
			branch:      "feature-x",
			excludePath: "",
			expected:    &Worktree{Path: "/path/to/worktree1", Branch: "feature-x"},
		},
		{
			name:        "branch not found",
			branch:      "nonexistent",
			excludePath: "",
			expected:    nil,
		},
		{
			name:        "exclude current path",
			branch:      "main",
			excludePath: "/path/to/repo",
			expected:    nil,
		},
		{
			name:        "find branch excluding different path",
			branch:      "main",
			excludePath: "/path/to/worktree1",
			expected:    &Worktree{Path: "/path/to/repo", Branch: "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindWorktreeByBranch(worktrees, tt.branch, tt.excludePath)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Errorf("expected %+v, got nil", tt.expected)
				return
			}
			if result.Path != tt.expected.Path || result.Branch != tt.expected.Branch {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}
