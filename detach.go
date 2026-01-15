package wtdetach

import (
	"fmt"
)

const (
	// DefaultSuffix is the default suffix for temporary branches
	DefaultSuffix = "__wt_detach"
)

// Options holds the command options
type Options struct {
	DryRun bool
	Revert bool
	Force  bool
	Yes    bool
}

// Result represents the result of an operation
type Result struct {
	Success      bool
	Message      string
	WorktreePath string
	TempBranch   string
}

// Detacher handles the detach/revert operations
type Detacher struct {
	git    *Git
	suffix string
}

// NewDetacher creates a new Detacher
func NewDetacher() *Detacher {
	return &Detacher{
		git:    &Git{},
		suffix: DefaultSuffix,
	}
}

// SetSuffix sets the suffix for temporary branches
func (d *Detacher) SetSuffix(suffix string) {
	if suffix != "" {
		d.suffix = suffix
	}
}

// GetSuffix returns the current suffix
func (d *Detacher) GetSuffix() string {
	return d.suffix
}

// LoadSuffixFromConfig loads the suffix from git config
func (d *Detacher) LoadSuffixFromConfig() {
	if suffix, err := d.git.Run("config", "--get", "wt-detach.suffix"); err == nil && suffix != "" {
		d.suffix = suffix
	}
}

// TempBranchName returns the temporary branch name for a given branch
func (d *Detacher) TempBranchName(branch string) string {
	return branch + d.suffix
}

// BranchExists checks if a branch exists
func (d *Detacher) BranchExists(branch string) bool {
	_, err := d.git.Run("rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// GetCurrentWorktreePath returns the path of the current worktree
func (d *Detacher) GetCurrentWorktreePath() (string, error) {
	path, err := d.git.Run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get current worktree path: %w", err)
	}
	return path, nil
}

// ListWorktrees returns a list of all worktrees
func (d *Detacher) ListWorktrees() ([]Worktree, error) {
	output, err := d.git.Run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	return ParseWorktreeList(output), nil
}

// FindWorktreeForBranch finds a worktree that has the specified branch checked out
// It excludes the current worktree
func (d *Detacher) FindWorktreeForBranch(branch string) (*Worktree, error) {
	currentPath, err := d.GetCurrentWorktreePath()
	if err != nil {
		return nil, err
	}

	worktrees, err := d.ListWorktrees()
	if err != nil {
		return nil, err
	}

	return FindWorktreeByBranch(worktrees, branch, currentPath), nil
}

// HasUncommittedChanges checks if a worktree has uncommitted changes
func (d *Detacher) HasUncommittedChanges(worktreePath string) bool {
	output, err := d.git.RunInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return true // Be safe on error
	}
	return output != ""
}

// CreateBranch creates a new branch at the current HEAD of a worktree
func (d *Detacher) CreateBranch(branch, worktreePath string) error {
	if _, err := d.git.RunInDir(worktreePath, "branch", branch); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", branch, err)
	}
	return nil
}

// DeleteBranch deletes a branch
func (d *Detacher) DeleteBranch(branch string) error {
	if _, err := d.git.Run("branch", "-D", branch); err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", branch, err)
	}
	return nil
}

// Checkout checks out a branch in a worktree
func (d *Detacher) Checkout(worktreePath, branch string) error {
	if _, err := d.git.RunInDir(worktreePath, "checkout", branch); err != nil {
		return fmt.Errorf("failed to checkout '%s' in '%s': %w", branch, worktreePath, err)
	}
	return nil
}

// Detach performs the detach operation
func (d *Detacher) Detach(branch string, opts *Options) (*Result, error) {
	if !d.BranchExists(branch) {
		return nil, fmt.Errorf("branch '%s' does not exist", branch)
	}

	tmpBranch := d.TempBranchName(branch)

	wt, err := d.FindWorktreeForBranch(branch)
	if err != nil {
		return nil, err
	}

	if wt == nil {
		return &Result{
			Success: true,
			Message: fmt.Sprintf("Branch '%s' is not checked out in any other worktree", branch),
		}, nil
	}

	if d.HasUncommittedChanges(wt.Path) {
		if !opts.Force {
			return nil, fmt.Errorf("uncommitted changes found in worktree: %s\n  Use --force to override", wt.Path)
		}
	}

	if d.BranchExists(tmpBranch) {
		return nil, fmt.Errorf("temporary branch '%s' already exists. Use --revert first or delete the branch manually", tmpBranch)
	}

	if opts.DryRun {
		return &Result{
			Success:      true,
			Message:      "dry-run",
			WorktreePath: wt.Path,
			TempBranch:   tmpBranch,
		}, nil
	}

	if err := d.CreateBranch(tmpBranch, wt.Path); err != nil {
		return nil, err
	}

	if err := d.Checkout(wt.Path, tmpBranch); err != nil {
		d.DeleteBranch(tmpBranch)
		return nil, err
	}

	return &Result{
		Success:      true,
		Message:      fmt.Sprintf("Branch '%s' detached successfully", branch),
		WorktreePath: wt.Path,
		TempBranch:   tmpBranch,
	}, nil
}

// Revert performs the revert operation
func (d *Detacher) Revert(branch string, opts *Options) (*Result, error) {
	if !d.BranchExists(branch) {
		return nil, fmt.Errorf("branch '%s' does not exist", branch)
	}

	tmpBranch := d.TempBranchName(branch)

	if !d.BranchExists(tmpBranch) {
		return nil, fmt.Errorf("temporary branch '%s' does not exist", tmpBranch)
	}

	wt, err := d.FindWorktreeForBranch(tmpBranch)
	if err != nil {
		return nil, err
	}

	if wt == nil {
		if opts.DryRun {
			return &Result{
				Success:    true,
				Message:    "dry-run: would delete branch",
				TempBranch: tmpBranch,
			}, nil
		}

		if err := d.DeleteBranch(tmpBranch); err != nil {
			return nil, err
		}

		return &Result{
			Success:    true,
			Message:    fmt.Sprintf("Deleted temporary branch '%s'", tmpBranch),
			TempBranch: tmpBranch,
		}, nil
	}

	if d.HasUncommittedChanges(wt.Path) {
		if !opts.Force {
			return nil, fmt.Errorf("uncommitted changes found in worktree: %s\n  Use --force to override", wt.Path)
		}
	}

	if opts.DryRun {
		return &Result{
			Success:      true,
			Message:      "dry-run",
			WorktreePath: wt.Path,
			TempBranch:   tmpBranch,
		}, nil
	}

	if err := d.Checkout(wt.Path, branch); err != nil {
		return nil, err
	}

	if err := d.DeleteBranch(tmpBranch); err != nil {
		return nil, err
	}

	return &Result{
		Success:      true,
		Message:      fmt.Sprintf("Branch '%s' restored successfully", branch),
		WorktreePath: wt.Path,
		TempBranch:   tmpBranch,
	}, nil
}
