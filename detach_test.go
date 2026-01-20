package wtdetach

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// resolvePath resolves symlinks in path (needed for macOS where /var -> /private/var)
func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

// setupTestRepo creates a test git repository with an initial commit
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := resolvePath(t, t.TempDir())

	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %v: %v\n%s", args, err, out)
		}
	}

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmds = [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "Initial commit"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %v: %v\n%s", args, err, out)
		}
	}

	return dir
}

// createBranch creates a new branch in the repo
func createBranch(t *testing.T, repoDir, branch string) {
	t.Helper()
	cmd := exec.Command("git", "branch", branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create branch %s: %v\n%s", branch, err, out)
	}
}

// createWorktree creates a worktree for the given branch
func createWorktree(t *testing.T, repoDir, worktreePath, branch string) {
	t.Helper()
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create worktree: %v\n%s", err, out)
	}
}

// getCurrentBranch returns the current branch of a directory
func getCurrentBranch(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// branchExistsInRepo checks if a branch exists in the repo
func branchExistsInRepo(t *testing.T, repoDir, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/heads/"+branch)
	cmd.Dir = repoDir
	return cmd.Run() == nil
}

// createUncommittedChange creates an uncommitted change in the directory
func createUncommittedChange(t *testing.T, dir string) {
	t.Helper()
	testFile := filepath.Join(dir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted change\n"), 0644); err != nil {
		t.Fatalf("failed to create uncommitted file: %v", err)
	}
}

func TestIntegration_DetachAndRevert(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)

	// Create a feature branch
	createBranch(t, repoDir, "feature-x")

	// Create a worktree for feature-x
	worktreeDir := filepath.Join(resolvePath(t, t.TempDir()), "worktree-feature-x")
	createWorktree(t, repoDir, worktreeDir, "feature-x")

	// Verify worktree is on feature-x
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-x" {
		t.Fatalf("worktree should be on feature-x, got %s", branch)
	}

	// Change to repo directory for the test
	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	// Create detacher
	d := NewDetacher()

	// Test: Detach feature-x
	result, err := d.Detach("feature-x", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Detach failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Detach should succeed")
	}
	if result.WorktreePath != worktreeDir {
		t.Errorf("WorktreePath: expected %s, got %s", worktreeDir, result.WorktreePath)
	}

	// Verify: worktree should now be on feature-x__wt_detach
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-x__wt_detach" {
		t.Errorf("worktree should be on feature-x__wt_detach, got %s", branch)
	}

	// Verify: temp branch exists
	if !branchExistsInRepo(t, repoDir, "feature-x__wt_detach") {
		t.Error("temp branch should exist")
	}

	// Test: Revert
	result, err = d.Revert("feature-x", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("Revert should succeed")
	}

	// Verify: worktree should be back on feature-x
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-x" {
		t.Errorf("worktree should be on feature-x, got %s", branch)
	}

	// Verify: temp branch should be deleted
	if branchExistsInRepo(t, repoDir, "feature-x__wt_detach") {
		t.Error("temp branch should be deleted")
	}
}

func TestIntegration_DetachWithUncommittedChanges(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-y")
	worktreeDir := filepath.Join(resolvePath(t, t.TempDir()), "worktree-feature-y")
	createWorktree(t, repoDir, worktreeDir, "feature-y")

	// Create uncommitted change in worktree
	createUncommittedChange(t, worktreeDir)

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Detach should fail without --force
	_, err := d.Detach("feature-y", &Options{Yes: true})
	if err == nil {
		t.Fatal("Detach should fail with uncommitted changes")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("error should mention uncommitted changes: %v", err)
	}

	// Verify: worktree should still be on feature-y
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-y" {
		t.Errorf("worktree should still be on feature-y, got %s", branch)
	}

	// Test: Detach should succeed with --force
	result, err := d.Detach("feature-y", &Options{Yes: true, Force: true})
	if err != nil {
		t.Fatalf("Detach with force should succeed: %v", err)
	}
	if !result.Success {
		t.Fatal("Detach with force should succeed")
	}

	// Verify: worktree should now be on temp branch
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-y__wt_detach" {
		t.Errorf("worktree should be on feature-y__wt_detach, got %s", branch)
	}
}

func TestIntegration_DetachBranchNotInWorktree(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "unused-branch")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Detach should return success with message that branch is not in any worktree
	result, err := d.Detach("unused-branch", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Detach should not error: %v", err)
	}
	if !result.Success {
		t.Fatal("Detach should succeed")
	}
	if !strings.Contains(result.Message, "not checked out") {
		t.Errorf("message should indicate branch is not in worktree: %s", result.Message)
	}
}

func TestIntegration_DetachNonexistentBranch(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Detach nonexistent branch should fail
	_, err := d.Detach("nonexistent", &Options{Yes: true})
	if err == nil {
		t.Fatal("Detach should fail for nonexistent branch")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention branch does not exist: %v", err)
	}
}

func TestIntegration_DryRun(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-dry")
	worktreeDir := filepath.Join(resolvePath(t, t.TempDir()), "worktree-feature-dry")
	createWorktree(t, repoDir, worktreeDir, "feature-dry")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Dry run should not make any changes
	result, err := d.Detach("feature-dry", &Options{DryRun: true})
	if err != nil {
		t.Fatalf("Dry run should not error: %v", err)
	}
	if result.Message != "dry-run" {
		t.Errorf("message should be 'dry-run': %s", result.Message)
	}

	// Verify: worktree should still be on feature-dry
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-dry" {
		t.Errorf("worktree should still be on feature-dry, got %s", branch)
	}

	// Verify: temp branch should not exist
	if branchExistsInRepo(t, repoDir, "feature-dry__wt_detach") {
		t.Error("temp branch should not be created in dry-run")
	}
}

func TestIntegration_TempBranchAlreadyExists(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-conflict")
	createBranch(t, repoDir, "feature-conflict__wt_detach") // Pre-create temp branch
	worktreeDir := filepath.Join(resolvePath(t, t.TempDir()), "worktree-conflict")
	createWorktree(t, repoDir, worktreeDir, "feature-conflict")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Detach should fail because temp branch already exists
	_, err := d.Detach("feature-conflict", &Options{Yes: true})
	if err == nil {
		t.Fatal("Detach should fail when temp branch already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention temp branch already exists: %v", err)
	}
}

func TestIntegration_CustomSuffix(t *testing.T) {
	// Setup
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-custom")
	worktreeDir := filepath.Join(resolvePath(t, t.TempDir()), "worktree-custom")
	createWorktree(t, repoDir, worktreeDir, "feature-custom")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()
	d.SetSuffix("__custom_suffix")

	// Test: Detach with custom suffix
	result, err := d.Detach("feature-custom", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Detach failed: %v", err)
	}

	// Verify: temp branch should use custom suffix
	if result.TempBranch != "feature-custom__custom_suffix" {
		t.Errorf("temp branch should use custom suffix: %s", result.TempBranch)
	}

	// Verify: worktree should be on custom suffixed branch
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-custom__custom_suffix" {
		t.Errorf("worktree should be on feature-custom__custom_suffix, got %s", branch)
	}

	// Test: Revert with custom suffix
	result, err = d.Revert("feature-custom", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}

	// Verify: worktree should be back on original branch
	if branch := getCurrentBranch(t, worktreeDir); branch != "feature-custom" {
		t.Errorf("worktree should be on feature-custom, got %s", branch)
	}
}

func TestIntegration_RevertWithoutWorktree(t *testing.T) {
	// Setup: Create temp branch but it's not checked out anywhere
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-orphan")
	createBranch(t, repoDir, "feature-orphan__wt_detach")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Revert should just delete the temp branch
	result, err := d.Revert("feature-orphan", &Options{Yes: true})
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if !result.Success {
		t.Fatal("Revert should succeed")
	}

	// Verify: temp branch should be deleted
	if branchExistsInRepo(t, repoDir, "feature-orphan__wt_detach") {
		t.Error("temp branch should be deleted")
	}
}

func TestDetach_RevertTempBranchNotFound(t *testing.T) {
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "feature-no-temp")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Revert should fail when temp branch doesn't exist
	_, err := d.Revert("feature-no-temp", &Options{Yes: true})
	if err == nil {
		t.Fatal("Revert should fail when temp branch doesn't exist")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention temp branch does not exist: %v", err)
	}
}

func TestDetach_RevertBranchNotFound(t *testing.T) {
	repoDir := setupTestRepo(t)

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// Test: Revert should fail when original branch doesn't exist
	_, err := d.Revert("nonexistent", &Options{Yes: true})
	if err == nil {
		t.Fatal("Revert should fail when branch doesn't exist")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention branch does not exist: %v", err)
	}
}

func TestDetach_BranchExists(t *testing.T) {
	repoDir := setupTestRepo(t)
	createBranch(t, repoDir, "test-branch")

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	if !d.BranchExists("test-branch") {
		t.Error("BranchExists should return true for existing branch")
	}
	if !d.BranchExists("main") {
		t.Error("BranchExists should return true for main branch")
	}
	if d.BranchExists("nonexistent") {
		t.Error("BranchExists should return false for nonexistent branch")
	}
}

func TestDetach_TempBranchName(t *testing.T) {
	d := NewDetacher()

	if name := d.TempBranchName("feature-x"); name != "feature-x__wt_detach" {
		t.Errorf("TempBranchName: expected feature-x__wt_detach, got %s", name)
	}

	d.SetSuffix("__custom")
	if name := d.TempBranchName("feature-x"); name != "feature-x__custom" {
		t.Errorf("TempBranchName with custom suffix: expected feature-x__custom, got %s", name)
	}
}

func TestDetach_GetUncommittedFiles(t *testing.T) {
	repoDir := setupTestRepo(t)

	oldWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(oldWd)

	d := NewDetacher()

	// No uncommitted files
	files := d.GetUncommittedFiles(repoDir)
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}

	// Create uncommitted files
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(repoDir, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("test\n"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	files = d.GetUncommittedFiles(repoDir)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), files)
	}
}

func TestFormatUncommittedError(t *testing.T) {
	// Test with few files
	err := formatUncommittedError("/path/to/worktree", []string{"file1.txt", "file2.txt"})
	errMsg := err.Error()

	if !strings.Contains(errMsg, "/path/to/worktree") {
		t.Errorf("error should contain worktree path: %s", errMsg)
	}
	if !strings.Contains(errMsg, "file1.txt") {
		t.Errorf("error should contain file1.txt: %s", errMsg)
	}
	if !strings.Contains(errMsg, "file2.txt") {
		t.Errorf("error should contain file2.txt: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Use --force to override") {
		t.Errorf("error should contain force hint: %s", errMsg)
	}

	// Test with more than 10 files
	manyFiles := make([]string, 15)
	for i := 0; i < 15; i++ {
		manyFiles[i] = "file" + string(rune('a'+i)) + ".txt"
	}
	err = formatUncommittedError("/path/to/worktree", manyFiles)
	errMsg = err.Error()

	if !strings.Contains(errMsg, "15 files or more") {
		t.Errorf("error should mention '15 files or more': %s", errMsg)
	}
	// Should NOT list individual files
	if strings.Contains(errMsg, "filea.txt") {
		t.Errorf("error should not list individual files when > 10: %s", errMsg)
	}
}
