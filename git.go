package wtdetach

import (
	"os/exec"
	"strings"
)

// Git provides methods to execute git commands
type Git struct{}

// Run executes a git command and returns the output
func (g *Git) Run(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// RunInDir executes a git command in a specific directory and returns the output
func (g *Git) RunInDir(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}
