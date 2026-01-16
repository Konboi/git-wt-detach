package wtdetach

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// CLI defines the command-line interface
type CLI struct {
	Branch  string           `arg:"" optional:"" help:"Branch name to detach or revert."`
	DryRun  bool             `help:"Show what would be done without making changes." short:"n"`
	Revert  bool             `help:"Revert the temporary detach." short:"r"`
	Force   bool             `help:"Force execution even with uncommitted changes." short:"f"`
	Yes     bool             `help:"Skip confirmation prompt." short:"y"`
	Init    string           `help:"Output shell completion script (bash, zsh, fish)." placeholder:"SHELL"`
	Version kong.VersionFlag `help:"Show version."`
}

// Run executes the CLI command
func (c *CLI) Run() error {
	if c.Init != "" {
		script, err := CompletionScript(c.Init)
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	}

	if c.Branch == "" {
		return fmt.Errorf("branch name is required")
	}

	d := NewDetacher()
	d.LoadSuffixFromConfig()

	opts := &Options{
		DryRun: c.DryRun,
		Revert: c.Revert,
		Force:  c.Force,
		Yes:    c.Yes,
	}

	if c.Revert {
		return c.runRevert(d, opts)
	}
	return c.runDetach(d, opts)
}

func (c *CLI) runDetach(d *Detacher, opts *Options) error {
	branch := c.Branch

	if !d.BranchExists(branch) {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	wt, err := d.FindWorktreeForBranch(branch)
	if err != nil {
		return err
	}

	if wt == nil {
		fmt.Printf("Branch '%s' is not checked out in any other worktree.\n", branch)
		return nil
	}

	fmt.Printf("✔ Found worktree: %s\n", wt.Path)

	if d.HasUncommittedChanges(wt.Path) {
		if !opts.Force {
			return fmt.Errorf("uncommitted changes found in worktree: %s\n  Use --force to override", wt.Path)
		}
		fmt.Printf("⚠ Warning: Uncommitted changes found in worktree: %s\n", wt.Path)
	}

	tmpBranch := d.TempBranchName(branch)

	if opts.DryRun {
		fmt.Printf("would create branch: %s\n", tmpBranch)
		fmt.Printf("would checkout in worktree: %s\n", wt.Path)
		return nil
	}

	if !opts.Yes {
		if !c.confirm(branch, wt.Path, tmpBranch) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	result, err := d.Detach(branch, opts)
	if err != nil {
		return err
	}

	fmt.Printf("✔ Created temp branch: %s\n", result.TempBranch)
	fmt.Printf("✔ Switched worktree branch\n")
	fmt.Printf("✔ Branch detached: %s\n", branch)
	return nil
}

func (c *CLI) runRevert(d *Detacher, opts *Options) error {
	branch := c.Branch
	tmpBranch := d.TempBranchName(branch)

	if !d.BranchExists(tmpBranch) {
		return fmt.Errorf("temporary branch '%s' does not exist", tmpBranch)
	}

	wt, err := d.FindWorktreeForBranch(tmpBranch)
	if err != nil {
		return err
	}

	if wt == nil {
		if opts.DryRun {
			fmt.Printf("would delete branch: %s\n", tmpBranch)
			return nil
		}

		result, err := d.Revert(branch, opts)
		if err != nil {
			return err
		}
		fmt.Printf("✔ Deleted temp branch: %s\n", result.TempBranch)
		return nil
	}

	fmt.Printf("✔ Found worktree with temp branch: %s\n", wt.Path)

	if d.HasUncommittedChanges(wt.Path) {
		if !opts.Force {
			return fmt.Errorf("uncommitted changes found in worktree: %s\n  Use --force to override", wt.Path)
		}
		fmt.Printf("⚠ Warning: Uncommitted changes found in worktree: %s\n", wt.Path)
	}

	if opts.DryRun {
		fmt.Printf("would checkout branch in worktree: %s -> %s\n", wt.Path, branch)
		fmt.Printf("would delete branch: %s\n", tmpBranch)
		return nil
	}

	if !opts.Yes {
		fmt.Printf("Worktree '%s' will be switched back to branch '%s'\n", wt.Path, branch)
		fmt.Printf("Temporary branch '%s' will be deleted.\n\n", tmpBranch)
		fmt.Print("Proceed? [y/N] ")
		if !readYesNo() {
			fmt.Println("Aborted.")
			return nil
		}
	}

	result, err := d.Revert(branch, opts)
	if err != nil {
		return err
	}

	fmt.Printf("✔ Switched worktree to: %s\n", branch)
	fmt.Printf("✔ Deleted temp branch: %s\n", result.TempBranch)
	fmt.Printf("✔ Branch restored: %s\n", branch)
	return nil
}

func (c *CLI) confirm(branch, worktreePath, tmpBranch string) bool {
	fmt.Printf("Branch '%s' is currently checked out in:\n", branch)
	fmt.Printf("  %s\n\n", worktreePath)
	fmt.Printf("It will be temporarily replaced by:\n")
	fmt.Printf("  %s\n\n", tmpBranch)
	fmt.Print("Proceed? [y/N] ")
	return readYesNo()
}

func readYesNo() bool {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
