---
phase: 5
title: "Agent DX & Safety Guardrails"
status: pending
priority: P1
effort: "2h"
dependencies: ["phase-02-client-engine-scan.md", "phase-03-context-commands.md", "phase-04-key-browser-inspect.md"]
---

# Phase 05: Agent DX & Safety Guardrails

## Overview
Optimizes the CLI for autonomous AI coding agents, background subagents, and CI pipelines. Implements compact single-line JSON formatting, surgical field filtering (`--fields`), dedicated `redis summary` health snapshot (< 500 bytes JSON), semantic exit codes (0 to 5) with structured JSON error messages on stderr, and destructive action guardrails (`pkg/safety/` enforcing `--force`, streaming chunked UNLINK operations, and supporting `--dry-run`).

## Requirements
- Functional:
  - `--json` / `--format=json`: Auto-compact 1-line JSON when output is piped or `--compact` is passed.
  - Stream Purity: Suppress standard loggers (`log.SetOutput(io.Discard)`) and route internal client warnings to stderr when `--json` is active, guaranteeing stdout contains 100% parseable JSON.
  - `--fields <list>`: Filter JSON output to only include specified keys (e.g. `--fields=key,type,ttl`).
  - `redis summary`: Compact health snapshot extracting ~15 vital indicators (Memory, Clients, DB keys, Hit rate, Role) in < 500 bytes JSON.
  - Semantic Exit Codes:
    - `0`: Success (`EXIT_OK`)
    - `1`: Socket failure/timeout (`EXIT_CONNECTION_ERROR`)
    - `2`: Authentication failure (`EXIT_AUTH_ERROR`)
    - `3`: Key not found (`EXIT_KEY_NOT_FOUND`)
    - `4`: Missing `--force` on destructive action (`EXIT_SAFETY_VIOLATION`)
    - `5`: Syntax/flag error (`EXIT_SYNTAX_ERROR`)
  - Destructive Guardrails (`redis del`):
    - Implement typed `SafetyError` with error code; avoid direct `os.Exit` in library code to ensure unit tests execute cleanly.
    - When `isatty` is true and `--force` is absent: Interactive prompt.
    - When `isatty` is false (Agent/CI) and `--force` is absent: Return `SafetyError{Code: 4}` + JSON error on stderr.
    - Stream pattern deletions in chunks of 500 keys to `UNLINK` immediately to avoid buffering millions of keys in RAM.
    - When `--dry-run` is present: Use running counters for matching keys and memory footprint without buffering key names.
- Non-functional:
  - Zero hanging processes (no blocked stdin reads in non-interactive mode).

## Architecture
```text
CLI Invocation
      │
      ▼
pkg/safety/guard.go:
  - Intercept Destructive Action
  - If --dry-run: stream count & return JSON summary
  - If !isatty.IsTerminal() && !--force:
      return &SafetyError{Code: 4, Message: "destructive action requires --force"}

cmd/ (or main.go):
  - Catch *SafetyError -> emit JSON to Stderr -> os.Exit(err.Code)

pkg/format/agent.go:
  - Suppress logs: log.SetOutput(io.Discard)
  - SerializeCompactJSON(data, fields) -> stdout
```

## Related Code Files
- Create: `cmd/summary.go`
- Create: `cmd/del.go`
- Create: `pkg/safety/guard.go`
- Create: `pkg/safety/errors.go`
- Create: `pkg/safety/errors_test.go`
- Create: `pkg/format/agent.go`
- Create: `pkg/safety/guard_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Implement Semantic Exit Codes & Structured Error Types
- **Files**: `pkg/safety/errors.go`, `pkg/safety/errors_test.go`
- **Action**: Define error constants (`ExitOK`, `ExitConnectionError`, `ExitAuthError`, `ExitKeyNotFound`, `ExitSafetyViolation`, `ExitSyntaxError`). Implement `type SafetyError struct { Code int; Message string }` and `EmitError(err error, code int)`.
- **Verification Command**: `go test -v -run TestSemanticErrors ./pkg/safety/...`
- **Expected Output**: PASS verifying error code mappings and JSON error formatting.

### Step 2: Implement Safety Guardrail with Chunked UNLINK & Dry-Run
- **Files**: `pkg/safety/guard.go`, `pkg/safety/guard_test.go`, `cmd/del.go`
- **Action**: Build `VerifyDestructiveAction(cmd *cobra.Command, target string) error` returning `SafetyError`. In `cmd/del.go`, stream pattern-matched keys in batches of 500 directly to `UNLINK`. For `--dry-run`, maintain integer counters.
- **Verification Command**: `go test -v -run TestSafetyGuardrail ./pkg/safety/...`
- **Expected Output**: PASS verifying exit code 4 error return on simulated non-TTY.

### Step 3: Implement `redis summary` Health Snapshot Command
- **Files**: `cmd/summary.go`
- **Action**: Parse Redis `INFO` & `DBSIZE` into a clean, compact struct (< 500 bytes JSON). Support `--fields` and `--format=json`.
- **Verification Command**: `go run main.go summary --json | jq .`
- **Expected Output**: Valid JSON object containing server metrics.

### Step 4: Implement Token-Optimized JSON Serializer & Stream Purifier
- **Files**: `pkg/format/agent.go`
- **Action**: Implement compact single-line JSON emitter with dynamic field filtering. Suppress default loggers on `--json`. Seed test keys before running show verification:
  ```bash
  docker exec capstone-redis-common redis-cli MSET test:u1 "Alice" test:u2 "Bob"
  ```
- **Verification Command**: `go run main.go show --json --fields=key,type --limit=3`
- **Expected Output**: Compact 1-line JSON array without indentation spaces or log pollution.

## Success Criteria
- [ ] `redis summary --json` returns valid JSON under 500 bytes.
- [ ] `redis show --json --fields=key,type` slashes token count by > 65%.
- [ ] `redis del "test:*" </dev/null` returns exit code 4 and JSON error on stderr without crashing tests.
- [ ] Pattern deletion streams in 500-key chunks to `UNLINK` without memory exhaustion.

## Risk Assessment
- **Risk**: Logging statements polluting stdout JSON stream.
- **Mitigation**: Route all client logs to stderr and discard root logger output when `--json` is active.
