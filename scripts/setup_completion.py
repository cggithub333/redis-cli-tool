#!/usr/bin/env python3
"""
setup_completion.py: Shell autocompletion installer for redis CLI tool.
Supports Zsh and Bash on Linux / macOS.
Mirrors the completion architecture used in ~/.oci/bin/completion/INIT-oci-completion.sh.
"""

import os
import sys
from pathlib import Path

COMPLETION_SCRIPT = """# Redis CLI Tool completion for Zsh and Bash

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

# Top-level Redis CLI command definitions with descriptions
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

# Subcommands for 'redis context'
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

    # 1. Top-level commands
    if (( CURRENT == 2 )) && [[ "$words[2]" != -* ]]; then
        local -a top_cmds=()
        for cmd desc in ${(kv)_REDIS_TOP_LEVEL_COMMANDS}; do
            top_cmds+=("${cmd}:${desc}")
        done
        _describe -t commands "redis command" top_cmds
        return
    fi

    # 2. Subcommands for 'context'
    if [[ "${words[2]}" == "context" || "${words[2]}" == "ctx" ]] && (( CURRENT == 3 )) && [[ "$words[3]" != -* ]]; then
        local -a ctx_cmds=()
        for cmd desc in ${(kv)_REDIS_CONTEXT_COMMANDS}; do
            ctx_cmds+=("${cmd}:${desc}")
        done
        _describe -t commands "context subcommand" ctx_cmds
        return
    fi

    # 3. Dynamic Cobra completion for flags, context names, and deeper arguments
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
"""

def main():
    home = Path.home()
    target_dir = home / ".redis-cli" / "bin" / "completion"
    target_dir.mkdir(parents=True, exist_ok=True)

    target_file = target_dir / "INIT-redis-completion.sh"
    target_file.write_text(COMPLETION_SCRIPT, encoding="utf-8")
    target_file.chmod(0o755)

    print(f"\033[38;5;42m✓\033[0m Written completion script to: {target_file}")

    source_line = '\n# Redis CLI Tool completion\n[[ -f "$HOME/.redis-cli/bin/completion/INIT-redis-completion.sh" ]] && source "$HOME/.redis-cli/bin/completion/INIT-redis-completion.sh"\n'

    # Determine shell rc file
    shell = os.environ.get("SHELL", "")
    rc_files = []
    if "zsh" in shell:
        rc_files.append(home / ".zshrc")
    elif "bash" in shell:
        rc_files.append(home / ".bashrc")
    else:
        # Default: try both if they exist
        if (home / ".zshrc").exists():
            rc_files.append(home / ".zshrc")
        if (home / ".bashrc").exists():
            rc_files.append(home / ".bashrc")

    for rc in rc_files:
        if rc.exists():
            content = rc.read_text(encoding="utf-8", errors="ignore")
            if "INIT-redis-completion.sh" in content:
                print(f"\033[38;5;214m•\033[0m Already configured in {rc}")
            else:
                with rc.open("a", encoding="utf-8") as f:
                    f.write(source_line)
                print(f"\033[38;5;42m✓\033[0m Appended completion loader to {rc}")

    print("\n\033[38;5;209mRedis CLI completion is ready!\033[0m")
    print("Reload your shell now:")
    print("  \033[1;37msource ~/.zshrc\033[0m (or restart your terminal)\n")

if __name__ == "__main__":
    main()
