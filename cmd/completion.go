package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var completionShellFlag string

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell|install]",
	Short: "Generate or install shell autocompletion script",
	Long: `Generate or install autocompletion scripts for redis subcommands and context flags.

To install automatically:
  redis completion install

To load manually in your current shell session:
  # Zsh
  source <(redis completion zsh)

  # Bash
  source <(redis completion bash)
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell", "install"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(out)
		case "zsh":
			return cmd.Root().GenZshCompletion(out)
		case "fish":
			return cmd.Root().GenFishCompletion(out, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(out)
		case "install":
			return runCompletionInstall(cmd)
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.Flags().StringVar(&completionShellFlag, "shell", "", "Target shell (zsh, bash). Auto-detected if omitted.")
}

const completionScriptTemplate = `# Redis CLI Tool completion for Zsh and Bash

# 1. Native Bash completion fallback
if [ -n "${BASH_VERSION:-}" ]; then
    _redis_completion() {
        local words cword
        if type _get_comp_words_by_ref &>/dev/null; then
            _get_comp_words_by_ref -n "=:" words cword
        else
            words=("${COMP_WORDS[@]}")
            cword=${COMP_CWORD}
        fi
        local out
        out=$(redis __complete "${words[@]:1:$cword}" 2>/dev/null)
        local directive=${out##*:}
        out=${out%:*}
        COMPREPLY=($(compgen -W "$out" -- "${words[$cword]}"))
    }
    complete -F _redis_completion -o default redis
    return 0 2>/dev/null || exit 0
fi

# 2. Rich Zsh completion with descriptions
(( $+functions[compdef] )) || { autoload -U compinit && compinit -i; }

typeset -gA _REDIS_TOP_LEVEL_COMMANDS=(
    ["context"]="Manage Redis connection contexts (create, use, ls, rename, delete)"
    ["explore"]="Launch full-screen interactive FZF key explorer with live preview"
    ["show"]="Browse and scan keys with metadata (Type, TTL, Memory)"
    ["inspect"]="Inspect a key's metadata, encoding, and safely preview its value"
    ["get"]="Get string value of a key"
    ["set"]="Set string value of a key with optional TTL"
    ["del"]="Safely delete keys or patterns with chunked UNLINK"
    ["summary"]="Compact Redis health snapshot (< 500 bytes JSON)"
    ["completion"]="Generate or install autocompletion scripts for shells"
)

typeset -gA _REDIS_CONTEXT_COMMANDS=(
    ["create"]="Create a new context profile"
    ["use"]="Switch active context"
    ["ls"]="List all configured contexts with connection health status"
    ["current"]="Print active context name"
    ["rename"]="Rename an existing connection context"
    ["delete"]="Delete a context profile"
    ["export"]="Export context configuration as YAML"
    ["import"]="Import contexts from a YAML file (or '-' for stdin)"
)

_redis() {
    local -a completions
    local -a completions_with_descriptions
    local -a response

    if (( CURRENT == 2 )) && [[ "$words[2]" != -* ]]; then
        local -a top_cmds=()
        for cmd desc in ${(kv)_REDIS_TOP_LEVEL_COMMANDS}; do
            top_cmds+=("${cmd}:${desc}")
        done
        _describe -t commands "redis command" top_cmds
        return
    fi

    if [[ "${words[2]}" == "context" || "${words[2]}" == "ctx" ]] && (( CURRENT == 3 )) && [[ "$words[3]" != -* ]]; then
        local -a ctx_cmds=()
        for cmd desc in ${(kv)_REDIS_CONTEXT_COMMANDS}; do
            ctx_cmds+=("${cmd}:${desc}")
        done
        _describe -t commands "context subcommand" ctx_cmds
        return
    fi

    local redis_bin="${commands[redis]:-$HOME/.local/bin/redis}"
    [[ -x "$redis_bin" ]] || return 1

    local -a comp_args=()
    local i
    for (( i=2; i <= CURRENT; i++ )); do
        comp_args+=("${words[i]}")
    done

    response=("${(@f)$("$redis_bin" __complete "${comp_args[@]}" 2>/dev/null)}")

    for line in "${response[@]}"; do
        if [[ "$line" == :* ]]; then
            continue
        fi
        if [[ "$line" == *$'\t'* ]]; then
            local key="${line%%$'\t'*}"
            local desc="${line#*$'\t'}"
            completions_with_descriptions+=("${key}:${desc}")
        elif [[ -n "$line" ]]; then
            completions+=("$line")
        fi
    done

    if (( ${#completions_with_descriptions} > 0 )); then
        _describe -t commands "options" completions_with_descriptions
    fi

    if (( ${#completions} > 0 )); then
        compadd -a completions
    fi
}

compdef _redis redis
`

func runCompletionInstall(cmd *cobra.Command) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine user home directory: %w", err)
	}

	targetDir := filepath.Join(homeDir, ".redis-cli", "bin", "completion")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", targetDir, err)
	}

	targetFile := filepath.Join(targetDir, "INIT-redis-completion.sh")
	if err := os.WriteFile(targetFile, []byte(completionScriptTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write completion script to %q: %w", targetFile, err)
	}

	sourceSnippet := fmt.Sprintf("\n# Redis CLI Tool completion\n[[ -f \"$HOME/.redis-cli/bin/completion/INIT-redis-completion.sh\" ]] && source \"$HOME/.redis-cli/bin/completion/INIT-redis-completion.sh\"\n")

	shell := completionShellFlag
	if shell == "" {
		if strings.Contains(os.Getenv("SHELL"), "zsh") {
			shell = "zsh"
		} else {
			shell = "bash"
		}
	}

	var rcFile string
	if shell == "zsh" {
		rcFile = filepath.Join(homeDir, ".zshrc")
	} else {
		rcFile = filepath.Join(homeDir, ".bashrc")
	}

	// Check if already sourced
	existingContent, err := os.ReadFile(rcFile)
	if err == nil && strings.Contains(string(existingContent), "INIT-redis-completion.sh") {
		fmt.Fprintf(cmd.OutOrStdout(), "Completion script updated at %s\nShell configuration (%s) is already configured.\n", targetFile, rcFile)
		return nil
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to update %q: %w", rcFile, err)
	}
	defer f.Close()

	if _, err := f.WriteString(sourceSnippet); err != nil {
		return fmt.Errorf("failed to append to %q: %w", rcFile, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully installed completion to %s\nAdded loader to %s\nRestart your shell or run: source %s\n", targetFile, rcFile, rcFile)
	return nil
}
