package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	wtdetach "github.com/Konboi/git-wt-detach"
)

const version = "0.1.0"

func main() {
	opts := &wtdetach.Options{}

	flag.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be done without making changes")
	flag.BoolVar(&opts.Revert, "revert", false, "Revert the temporary detach")
	flag.BoolVar(&opts.Force, "force", false, "Force execution even with uncommitted changes")
	flag.BoolVar(&opts.Yes, "yes", false, "Skip confirmation prompt")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git wt-detach <branch> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Temporarily detach a branch checked out in another worktree.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("git-wt-detach version %s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	branch := flag.Arg(0)

	if err := run(branch, opts); err != nil {
		fmt.Fprintf(os.Stderr, "✖ %s\n", err)
		os.Exit(1)
	}
}

func run(branch string, opts *wtdetach.Options) error {
	d := wtdetach.NewDetacher()
	d.LoadSuffixFromConfig()

	if opts.Revert {
		return runRevert(d, branch, opts)
	}
	return runDetach(d, branch, opts)
}

func runDetach(d *wtdetach.Detacher, branch string, opts *wtdetach.Options) error {
	// Check if branch exists
	if !d.BranchExists(branch) {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	// Find worktree
	wt, err := d.FindWorktreeForBranch(branch)
	if err != nil {
		return err
	}

	if wt == nil {
		fmt.Printf("Branch '%s' is not checked out in any other worktree.\n", branch)
		return nil
	}

	fmt.Printf("✔ Found worktree: %s\n", wt.Path)

	// Check for uncommitted changes
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

	// Confirmation prompt
	if !opts.Yes {
		if !confirm(branch, wt.Path, tmpBranch) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Execute detach
	result, err := d.Detach(branch, opts)
	if err != nil {
		return err
	}

	fmt.Printf("✔ Created temp branch: %s\n", result.TempBranch)
	fmt.Printf("✔ Switched worktree branch\n")
	fmt.Printf("✔ Branch detached: %s\n", branch)
	return nil
}

func runRevert(d *wtdetach.Detacher, branch string, opts *wtdetach.Options) error {
	tmpBranch := d.TempBranchName(branch)

	// Check if temp branch exists
	if !d.BranchExists(tmpBranch) {
		return fmt.Errorf("temporary branch '%s' does not exist", tmpBranch)
	}

	// Find worktree with temp branch
	wt, err := d.FindWorktreeForBranch(tmpBranch)
	if err != nil {
		return err
	}

	if wt == nil {
		// Temp branch exists but not checked out
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

	// Check for uncommitted changes
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

	// Confirmation prompt
	if !opts.Yes {
		fmt.Printf("Worktree '%s' will be switched back to branch '%s'\n", wt.Path, branch)
		fmt.Printf("Temporary branch '%s' will be deleted.\n\n", tmpBranch)
		fmt.Print("Proceed? [y/N] ")
		if !readYesNo() {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Execute revert
	result, err := d.Revert(branch, opts)
	if err != nil {
		return err
	}

	fmt.Printf("✔ Switched worktree to: %s\n", branch)
	fmt.Printf("✔ Deleted temp branch: %s\n", result.TempBranch)
	fmt.Printf("✔ Branch restored: %s\n", branch)
	return nil
}

func confirm(branch, worktreePath, tmpBranch string) bool {
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
