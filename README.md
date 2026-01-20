# git-wt-detach

A Git subcommand that temporarily detaches a branch checked out in another worktree, making it available for checkout in the current repository.

## Overview

This tool works around Git's limitation that prevents the same branch from being checked out in multiple worktrees simultaneously.

### Use Cases

1. You want to checkout a branch in the main repository while preserving Bazel cache, but the branch is currently used in another worktree
2. You need to temporarily move a branch between worktrees in a multi-worktree setup

## Installation

```bash
go install github.com/Konboi/git-wt-detach/cmd/git-wt-detach@latest
```

Or build from source:

```bash
git clone https://github.com/Konboi/git-wt-detach.git
cd git-wt-detach
go build -o git-wt-detach ./cmd/git-wt-detach

# Place in your PATH
mv git-wt-detach /usr/local/bin/
```

## Usage

### Detach a branch

```bash
git wt-detach <branch>
```

When the specified branch is checked out in another worktree:
1. Creates a temporary branch (`<branch>__wt_detach`)
2. Switches the target worktree to the temporary branch
3. Makes the original branch available for checkout

### Revert the detach

```bash
git wt-detach <branch> --revert
```

1. Switches the target worktree back to the original branch
2. Deletes the temporary branch

### Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Show what would be done without making changes |
| `--force` | Force execution even with uncommitted changes |
| `--yes` | Skip confirmation prompt |
| `--revert` | Revert the temporary detach |
| `--checkout` | Checkout the branch after detaching |
| `--init` | Output shell completion script (bash, zsh, fish) |
| `--version` | Show version |

## Shell Integration

Enable tab completion for branch names:

**Zsh:**

```bash
eval "$(git-wt-detach --init zsh)"
```

Add to your `~/.zshrc` for persistent completion.

**Bash:**

```bash
eval "$(git-wt-detach --init bash)"
```

Add to your `~/.bashrc` for persistent completion.

**Fish:**

```bash
git-wt-detach --init fish | source
```

Add to your `~/.config/fish/config.fish` for persistent completion.

## Example

```bash
# feature-x is checked out in another worktree
$ git wt-detach feature-x
✔ Found worktree: ../repo-wt-feature
Branch 'feature-x' is currently checked out in:
  ../repo-wt-feature

It will be temporarily replaced by:
  feature-x__wt_detach

Proceed? [y/N] y
✔ Created temp branch: feature-x__wt_detach
✔ Switched worktree branch
✔ Branch detached: feature-x

# Now you can checkout feature-x in the current repository
$ git checkout feature-x

# Or use --checkout (-c) to detach and checkout in one step
$ git wt-detach feature-x -c
✔ Found worktree: ../repo-wt-feature
✔ Created temp branch: feature-x__wt_detach
✔ Switched worktree branch
✔ Branch detached: feature-x
✔ Checked out: feature-x

# When done, revert to original state
$ git wt-detach feature-x --revert
✔ Found worktree with temp branch: ../repo-wt-feature
✔ Switched worktree to: feature-x
✔ Deleted temp branch: feature-x__wt_detach
✔ Branch restored: feature-x
```

## Configuration

### Custom suffix for temporary branches

The default suffix is `__wt_detach`. You can change it via git config:

```bash
git config wt-detach.suffix "__tmp"
```

## Safety Features

- Fails if the target worktree has uncommitted changes (use `--force` to override)
  - Shows up to 10 uncommitted files in the error message
  - Shows "N files or more" when there are more than 10 uncommitted files
- Fails if the temporary branch already exists
- Use `--dry-run` to preview changes before execution

## Requirements

- Git 2.20+
- Go 1.21+ (for building)

## License

MIT
