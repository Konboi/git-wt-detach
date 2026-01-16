package wtdetach

import "fmt"

// CompletionScript returns the shell completion script for the specified shell
func CompletionScript(shell string) (string, error) {
	switch shell {
	case "bash":
		return bashCompletion, nil
	case "zsh":
		return zshCompletion, nil
	case "fish":
		return fishCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish)", shell)
	}
}

const bashCompletion = `# bash completion for git-wt-detach
_git_wt_detach_branches() {
    git worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's/^branch refs\/heads\///'
}

_git_wt_detach() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local branches=$(_git_wt_detach_branches)
    COMPREPLY=($(compgen -W "${branches}" -- "${cur}"))
}

# Complete for direct command
complete -F _git_wt_detach git-wt-detach

# Complete for git subcommand
_git_wt_detach_subcommand() {
    if [[ ${COMP_WORDS[1]} == "wt-detach" ]]; then
        local cur="${COMP_WORDS[COMP_CWORD]}"
        local branches=$(_git_wt_detach_branches)
        COMPREPLY=($(compgen -W "${branches}" -- "${cur}"))
    fi
}

# Hook into git completion if available
if type _git &>/dev/null; then
    _git_wt_detach_orig_git=$(declare -f _git | tail -n +2)
    _git() {
        if [[ ${COMP_WORDS[1]} == "wt-detach" ]]; then
            _git_wt_detach_subcommand
        else
            eval "${_git_wt_detach_orig_git}"
        fi
    }
fi
`

const zshCompletion = `# zsh completion for git-wt-detach
_git-wt-detach() {
    local -a branches
    branches=(${(f)"$(git worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's/^branch refs\/heads\///')"})
    _describe 'branch' branches
}

compdef _git-wt-detach git-wt-detach

# Also register as git subcommand
_git-wt-detach-subcommand() {
    local -a branches
    branches=(${(f)"$(git worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's/^branch refs\/heads\///')"})
    _describe 'branch' branches
}

# Register completion for "git wt-detach"
zstyle ':completion:*:*:git:*' user-commands wt-detach:'Temporarily detach a branch checked out in another worktree'
`

const fishCompletion = `# fish completion for git-wt-detach
function __fish_git_wt_detach_branches
    git worktree list --porcelain 2>/dev/null | grep '^branch ' | sed 's/^branch refs\/heads\///'
end

complete -c git-wt-detach -f -a '(__fish_git_wt_detach_branches)' -d 'Branch'
complete -c git-wt-detach -s n -l dry-run -d 'Show what would be done without making changes'
complete -c git-wt-detach -s r -l revert -d 'Revert the temporary detach'
complete -c git-wt-detach -s f -l force -d 'Force execution even with uncommitted changes'
complete -c git-wt-detach -s y -l yes -d 'Skip confirmation prompt'
complete -c git-wt-detach -l version -d 'Show version'

# git subcommand completion
complete -c git -n '__fish_git_using_command wt-detach' -f -a '(__fish_git_wt_detach_branches)' -d 'Branch'
`
