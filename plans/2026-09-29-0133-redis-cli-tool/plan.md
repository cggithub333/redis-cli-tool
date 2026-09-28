---
title: "Redis Multi-Context CLI Tool Implementation Plan"
status: pending
priority: P1
mode: auto
specPath: "/home/james/temp/brainstorms/redis-cli-tool/spec.md"
blockedBy: []
blocks: []
totalPhases: 7
createdAt: "2026-09-29T01:33:00+07:00"
updatedAt: "2026-09-29T01:39:00+07:00"
---

# Master Architecture Plan: Redis Multi-Context CLI Tool (`redis`)

A high-performance, single-binary CLI tool written in Go that delivers Docker/Kubernetes-style context management, safe non-blocking key exploration via `SCAN` pipelines, dual Human & AI Agent DX, resilient safety guardrails, an interactive full-screen TUI, and a self-contained Python E2E integration test suite utilizing isolated virtual `XDG_CONFIG_HOME` paths.

---

## 1. Architectural Blueprint & Target Deliverable

- **Binary Output**: `redis` (Single static binary, Go 1.23+)
- **Configuration Store**: `~/.config/redis/contexts.yaml` (0600 POSIX permissions, env var expansion, atomic rename writes, file locking)
- **Primary Dependencies**:
  - `github.com/spf13/cobra` (Command hierarchy & persistent flags)
  - `github.com/redis/go-redis/v9` (Connection pool, RESP3 protocol, pipelines)
  - `github.com/charmbracelet/lipgloss` (TrueColor terminal tables, badges, styles)
  - `github.com/charmbracelet/bubbletea` (Interactive full-screen TUI key explorer)
  - `github.com/charmbracelet/bubbles` (TUI tables, viewports, text inputs)
  - `github.com/mattn/go-isatty` (TTY vs pipe detection for smart safety gates)
  - `gopkg.in/yaml.v3` (Context configuration serialization)
- **Integration Test Runner**: Python 3 standard library `unittest` (`tests/test_cli_xdg.py`) with dynamic sandboxed `XDG_CONFIG_HOME` directories.

---

## 2. Phase Execution Matrix (Strictly Linear DAG)

| Phase | Title | Effort | Status | Key Deliverable |
| :---: | :--- | :---: | :---: | :--- |
| **01** | [Core Scaffolding & Context Store](phase-01-core-scaffolding-config.md) | 2h | `completed` | Go module, Cobra root, 3-tier resolver, 0600 YAML store, atomic writes, flock, env expansion |
| **02** | [Client Engine & Non-Blocking SCAN](phase-02-client-engine-scan.md) | 2h | `completed` | `go-redis` pool, embedded client, TLS, ping, chunked SCAN cursor batching + metadata pipelines |
| **03** | [Context Subcommands & Table Formatter](phase-03-context-commands.md) | 2h | `completed` | `redis context (ls, use, create, delete, current, export, import)`, `pkg/format/table.go`, non-TTY safety warning |
| **04** | [Key Exploration & Adaptive Decoder](phase-04-key-browser-inspect.md) | 2.5h | `completed` | `redis show/inspect/get/set`, safe iterators (HSCAN/LRANGE), auto-JSON highlight, hex-dump, 256KB cap |
| **05** | [Agent DX & Safety Guardrails](phase-05-agent-dx-safety-guardrails.md) | 2h | `completed` | `--json --compact`, `--fields`, `redis summary`, typed `SafetyError` (0-5), chunked UNLINK, clean stdout |
| **06** | [Interactive Full-Screen TUI](phase-06-interactive-tui-explore.md) | 2.5h | `completed` | `redis explore` powered by Bubble Tea & Lipgloss with keyboard navigation & non-TTY guardrail |
| **07** | [Python XDG Integration & Review Board](phase-07-review-board-audit.md) | 1.5h | `pending` | Python `tests/test_cli_xdg.py` sandbox suite, Docker log hygiene, Review Board quality audit |

---

## 3. Technology Stack & Design Contracts

```text
[Terminal / TTY]                [Pipe / AI Agent / CI]
        │                                  │
        ▼                                  ▼
┌────────────────────────────────────────────────────────┐
│               cmd/root.go (Cobra Router)               │
│ Global Flags: --context, --format, --json, --force     │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│     pkg/config/ (3-Tier Context Precedence Engine)     │
│   --context flag > REDIS_CONTEXT env > YAML config     │
│   (Atomic writes: .tmp + Sync + os.Rename + flock)     │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│           pkg/safety/ (Guardrails & Memory Cap)        │
│  - isatty check: require --force for non-interactive   │
│  - Return typed *SafetyError{Code: 4} (no os.Exit)     │
│  - 256KB preview ceiling to prevent CLI OOM            │
│  - Chunked UNLINK streaming (500 keys/batch)           │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│         pkg/client/ (go-redis/v9 Pool & Protocol)      │
│  - Embedded *redis.Client for downstream subcommands   │
│  - Chunked SCAN pipelines (max 100-500 keys per flush) │
└──────────────────────────┬─────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
   Docker Redis (:6379)       Docker Redis (:6380)
   capstone-redis-common      capstone-redis-realtime
```

---

## 4. Empirical Verification Commands

Inspect system health, compile the binary, run Go unit tests, and execute the Python sandboxed XDG integration suite:

```bash
# 1. Run all Go unit tests with race detection
go test -v -race ./pkg/...

# 2. Build standalone binary
go build -o bin/redis main.go

# 3. Run Python E2E integration test suite in isolated virtual XDG_CONFIG_HOME
python3 -m unittest discover -s tests -v

# 4. Manual CLI sanity check against live Docker Redis instances
./bin/redis context ls
./bin/redis summary --json | jq .
```
