---
phase: 1
title: "Core Scaffolding & Context Store"
status: completed
priority: P1
effort: "2h"
dependencies: []
---

# Phase 01: Core Scaffolding & Context Store

## Overview
Initializes the Go module, fetches core dependencies, scaffolds the minimal entrypoint and Cobra root command tree with global persistent flags, and implements the configuration management layer supporting POSIX 0600 file security, atomic file replacement (`.tmp` + `Sync()` + `os.Rename()`), file locking (`flock`), environment variable interpolation (`${REDIS_PASS}`), and 3-tier context precedence resolution.

## Requirements
- Functional:
  - Initialize Go module `redis-cli-tool`.
  - Fetch all core ecosystem dependencies (`cobra`, `go-redis/v9`, `lipgloss`, `bubbletea`, `bubbles`, `go-isatty`, `yaml.v3`).
  - Create standard XDG config path at `~/.config/redis/contexts.yaml`.
  - Enforce file permissions `0600` for `contexts.yaml` (read/write only by owner).
  - Implement atomic write replacement with `contexts.yaml.tmp` and `os.Rename()` to prevent corrupted config files on unexpected crashes.
  - Implement 3-tier context resolution: `--context <name>` flag > `REDIS_CONTEXT` env var > `current-context` in YAML.
  - Automatically interpolate environment variables in passwords/usernames (`${VAR}`).
- Non-functional:
  - Sub-millisecond context resolution without disk I/O bottlenecks.
  - 100% unit test coverage for config parser and precedence resolver.

## Architecture
```text
Root Flags (--context, --format, --json, --force)
               │
               ▼
pkg/config/resolver.go:
  1. Check flag: cmd.Flags().GetString("context")
  2. If empty, check os.Getenv("REDIS_CONTEXT")
  3. If empty, read store.CurrentContext from ~/.config/redis/contexts.yaml
               │
               ▼
pkg/config/store.go:
  - LoadYAML(path) with os.OpenFile(path, O_RDWR|O_CREATE, 0600)
  - SaveYAML(path):
      1. Open path + ".tmp" with 0600
      2. Write YAML & f.Sync()
      3. os.Rename(tmp, path)  <-- Atomic crash-safe replacement
  - ExpandEnvVars() across auth fields
```

## Related Code Files
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `pkg/config/context.go`
- Create: `pkg/config/store.go`
- Create: `pkg/config/resolver.go`
- Create: `pkg/config/store_test.go`
- Create: `pkg/config/resolver_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Initialize Go Module & Core Dependencies
- **Files**: `go.mod`, `main.go`, `cmd/root.go`
- **Action**: Run `go mod init redis-cli-tool`. Fetch all dependencies:
  ```bash
  go get github.com/spf13/cobra@latest \
         github.com/redis/go-redis/v9@latest \
         github.com/charmbracelet/lipgloss@latest \
         github.com/charmbracelet/bubbletea@latest \
         github.com/charmbracelet/bubbles@latest \
         github.com/mattn/go-isatty@latest \
         gopkg.in/yaml.v3@latest
  ```
  Scaffold minimal `cmd/root.go` and `main.go` importing Cobra to anchor dependencies.
- **Verification Command**: `go mod verify && go vet ./...`
- **Expected Output**: Exit code 0, clean module verification.

### Step 2: Implement Context Data Model & Atomic Storage with 0600 Permissions
- **Files**: `pkg/config/context.go`, `pkg/config/store.go`, `pkg/config/store_test.go`
- **Action**: Define `Context` and `Config` structs. Implement `Load()`, `Save()` using atomic temporary file renaming (`.tmp` -> `os.Rename`) and POSIX file locking (`syscall.Flock`). Implement environment variable expansion for secrets.
- **Verification Command**: `go test -v -run TestStore ./pkg/config/...`
- **Expected Output**: PASS verifying atomic save, permissions, and env expansion.

### Step 3: Implement 3-Tier Precedence Resolver
- **Files**: `pkg/config/resolver.go`, `pkg/config/resolver_test.go`
- **Action**: Implement `ResolveContext(flagVal string) (*Context, error)` verifying flag > env > config precedence.
- **Verification Command**: `go test -v -run TestResolveContext ./pkg/config/...`
- **Expected Output**: PASS verifying all 3 priority levels.

### Step 4: Scaffold Cobra Root Command Flags & Help System
- **Files**: `cmd/root.go`, `main.go`
- **Action**: Configure persistent flags on root command (`--context`, `--format`, `--json`, `--compact`, `--fields`, `--force`, `--timeout`).
- **Verification Command**: `go run main.go --help`
- **Expected Output**: Displays complete help menu with description and flags.

## Success Criteria
- [x] `go.mod` and `go.sum` contain all 7 core packages.
- [x] Context store creates and atomically writes `contexts.yaml` with 0600 permissions.
- [x] 3-tier resolver passes all test cases (flag, env, yaml).
- [x] `redis --help` runs and outputs usage instructions.

## Risk Assessment
- **Risk**: Interrupted file write corrupts configuration.
- **Mitigation**: Atomic `.tmp` write with `Sync()` and `os.Rename()` ensures zero partial-write corruption.
