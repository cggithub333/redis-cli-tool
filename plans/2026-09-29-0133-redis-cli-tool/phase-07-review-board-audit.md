---
phase: 7
title: "Python XDG Integration Suite & Review Board Quality Audit"
status: completed
priority: P1
effort: "1.5h"
dependencies:
  - "phase-01-core-scaffolding-config.md"
  - "phase-02-client-engine-scan.md"
  - "phase-03-context-commands.md"
  - "phase-04-key-browser-inspect.md"
  - "phase-05-agent-dx-safety-guardrails.md"
  - "phase-06-interactive-tui-explore.md"
---

# Phase 07: Python XDG Integration Suite & Review Board Quality Audit

## Overview
Concluding architectural quality and verification gate. Implements a self-contained Python E2E integration test suite (`tests/test_cli_xdg.py`) using Python's standard library `unittest` and isolated `XDG_CONFIG_HOME` temporary paths. Executes Stage 0 deterministic runtime checks (Docker container logs, full Go race-detection suite) and opens the Review Board adversarial code review pass.

## Requirements
- Functional:
  - Python E2E Test Suite (`tests/test_cli_xdg.py`):
    - Uses `tempfile.TemporaryDirectory` to dynamically redirect `XDG_CONFIG_HOME` for every test run, keeping user's host environment 100% pristine.
    - Tests context lifecycle: `create`, `ls`, `use`, `current`, `export`, `delete`.
    - Tests data commands: `set`, `get`, `show --format=table`, `show --json --compact --fields=key,type`.
    - Tests agent snapshot: `summary --json` asserting payload size < 500 bytes.
    - Tests safety guardrails: non-TTY `del` returns exit code 4, `--dry-run` counts keys without mutating state.
    - Tests stream purity: verifies stdout parses cleanly as JSON with 0 log messages.
  - Stage 0 Check 1: 0 dirty or crashing container logs across the running Docker stack (`docker logs --tail=100 capstone-redis-common`).
  - Stage 0 Check 2: 100% test pass rate across unit, integration, and CLI smoke suites (`go test -v -race ./...`).
  - Stage 0 Check 3: Deterministic binary build verified under Linux x86_64 (`go build -o bin/redis main.go`).
  - Review Board Panel: Multi-persona review via `/james:code-review --pending --board`.
- Non-functional:
  - 100% pass rate across both Go test suite and Python XDG integration suite.

## Architecture
```text
All Implementation Phases (01 - 06) Completed
                    │
                    ▼
Step 1: Python XDG Sandbox E2E Integration Suite (tests/test_cli_xdg.py)
  - Virtual XDG_CONFIG_HOME = /tmp/xdg_test_XXXXXX
  - Subprocess calls to ./bin/redis
  - Zero pollution of ~/.config/redis/contexts.yaml
                    │
                    ▼
Step 2: Docker Container Logs Gate:
  - docker logs capstone-redis-common --tail=20 (0 errors)
                    │
                    ▼
Step 3: Go Test Suite with Race Detection:
  - go test -v -race ./... (100% pass)
                    │
                    ▼
Step 4: Review Board Audit Gate:
  - git diff --check && go vet ./...
  - Handoff to: /james:code-review --pending --board
```

## Related Code Files
- Create: `tests/test_cli_xdg.py`
- Inspect: All code under `cmd/`, `pkg/`, `main.go`, `go.mod`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Implement Python XDG Integration Suite
- **Files**: `tests/test_cli_xdg.py`
- **Action**: Create end-to-end integration tests using `unittest` and `subprocess.run`. Sandbox `XDG_CONFIG_HOME` via `tempfile.TemporaryDirectory`. Assert on context commands, JSON output formatting, and semantic exit codes.
- **Verification Command**: `python3 -m unittest discover -s tests -v`
- **Expected Output**: OK (all test cases passing in isolated sandbox).

### Step 2: Run Stage 0 Docker Container Hygiene Check
- **Files**: External runtime
- **Action**: Inspect logs of `capstone-redis-common` and `capstone-redis-realtime` to verify zero crash loops or protocol desynchronization.
- **Verification Command**: `docker logs capstone-redis-common --tail=20 && docker logs capstone-redis-realtime --tail=20`
- **Expected Output**: Clean Redis operational logs with 0 fatal errors.

### Step 3: Run Full Go Test Suite with Race Detection
- **Files**: All packages
- **Action**: Run `go test -v -race ./...` and `go vet ./...`.
- **Verification Command**: `go test -v -race ./...`
- **Expected Output**: PASS across all Go packages, 0 race conditions.

### Step 4: Execute Deterministic Pre-Review Checks & Board Handoff
- **Files**: Repository diff
- **Action**: Run git diff checks and trigger the Review Board audit gate.
- **Verification Command**: `git diff --check && go vet ./...`
- **Expected Output**: Exit code 0, ready for `/james:code-review --pending --board`.

## Success Criteria
- [x] Python XDG integration suite passes 100% against the compiled `./bin/redis` binary.
- [x] 0 container crashes or dirty logs in Docker Redis instances.
- [x] 100% Go test suite passing with `-race` enabled.
- [x] Plan concludes cleanly with Review Board sign-off.

## Risk Assessment
- **Risk**: Python test runner missing `pytest`.
- **Mitigation**: Using Python's standard library `unittest` ensures zero external dependencies.
