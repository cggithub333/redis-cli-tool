---
phase: 3
title: "Context Management Subcommands & Table Formatter"
status: completed
priority: P1
effort: "2h"
dependencies: ["phase-01-core-scaffolding-config.md", "phase-02-client-engine-scan.md"]
---

# Phase 03: Context Management Subcommands & Table Formatter

## Overview
Implements the full suite of `redis context` subcommands mimicking Docker and Kubernetes context workflows. Creates the foundational Lipgloss table renderer (`pkg/format/table.go`) for ANSI color tables with badges. Supports listing, switching, creating, deleting, testing connectivity, securely exporting/importing context definitions, and issuing non-TTY concurrency advisories on global context switching.

## Requirements
- Functional:
  - Create foundational table formatter `pkg/format/table.go` with Lipgloss column alignment and status badges.
  - `redis context ls`: Display table of configured contexts, indicating active context with `*` and live status badge (`HEALTHY` / `UNREACHABLE`).
  - `redis context use <name>`: Switch the global active context in `contexts.yaml`. When executed without a TTY (`!isatty`), print an advisory to stderr recommending `--context <name>` or `REDIS_CONTEXT` for parallel subagents.
  - `redis context current`: Print active context name.
  - `redis context create <name> [flags]`: Add new context (flags: `--host`, `--port`, `--db`, `--password`, `--tls`).
  - `redis context delete <name>`: Remove context definition.
  - `redis context export [--include-secrets]`: Export sanitized YAML with password placeholders (`${NAME_PASSWORD}`) unless `--include-secrets` is specified.
  - `redis context import <file>`: Import and merge contexts into local store.
- Non-functional:
  - Pretty ANSI color output for terminal users and clean JSON for agent scripts.

## Architecture
```text
redis context ls ────────► pkg/config/store.go (List contexts)
                           pkg/client/client.go (Concurrent ping healthcheck)
                           pkg/format/table.go (Render Lipgloss table)

redis context export ────► pkg/config/sanitizer.go (Mask secrets to ${ENV})

redis context use ───────► Concurrency Guard:
                           If !isatty.IsTerminal():
                             Log advisory to Stderr: "Note: For concurrent agents, prefer --context <name>"
```

## Related Code Files
- Create: `pkg/format/table.go`
- Create: `pkg/format/table_test.go`
- Create: `cmd/context.go`
- Create: `cmd/context_test.go`
- Create: `pkg/config/sanitizer.go`
- Create: `pkg/config/sanitizer_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Implement Core Lipgloss Table Formatter
- **Files**: `pkg/format/table.go`, `pkg/format/table_test.go`
- **Action**: Implement generic table renderer supporting headers, rows, column padding, and colored status badges (`HEALTHY`=Green, `UNREACHABLE`=Red, `ACTIVE`=Cyan).
- **Verification Command**: `go test -v -run TestTableRenderer ./pkg/format/...`
- **Expected Output**: PASS verifying table rendering.

### Step 2: Implement Context Sanitizer
- **Files**: `pkg/config/sanitizer.go`, `pkg/config/sanitizer_test.go`
- **Action**: Implement `SanitizeContext(c *Context) *Context` masking passwords into `${NAME_PASSWORD}` placeholders.
- **Verification Command**: `go test -v -run TestSanitizer ./pkg/config/...`
- **Expected Output**: PASS verifying secrets are masked.

### Step 3: Implement Context Subcommands (create, use, ls, current, delete, export, import)
- **Files**: `cmd/context.go`, `cmd/context_test.go`
- **Action**: Implement Cobra subcommands under `contextCmd`. Concurrently ping instances during `ls` with 500ms timeout per target. Add non-TTY advisory on `use`.
- **Verification Command**: `go test -v -run TestContextCRUD ./cmd/...`
- **Expected Output**: PASS creating, listing, exporting, and deleting contexts.

## Success Criteria
- [x] `pkg/format/table.go` is established and reusable across CLI commands.
- [x] `redis context create local-common --host 127.0.0.1 --port 6379` successfully creates context.
- [x] `redis context ls` lists both `local-common` and `local-realtime` with health status badges.
- [x] `redis context use local-realtime` switches active context.
- [x] `redis context export` produces sanitized YAML without plaintext credentials.

## Risk Assessment
- **Risk**: Remote unreachable hosts slowing down `context ls`.
- **Mitigation**: Parallel goroutines with 500ms hard timeout per ping.
