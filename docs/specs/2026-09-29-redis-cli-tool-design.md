# Architectural Design Specification: Redis Multi-Context CLI Tool (`redis`)

- **Author**: JamesKit Architecture Brainstormer
- **Target Project**: `45-redis-cli-tool`
- **Output Binary**: `redis` (Go 1.23+)
- **Date**: `2026-09-29`
- **Status**: `APPROVED_BY_USER`

---

## 1. System Overview & Problem Statement

Modern backend engineering and cloud deployments rely heavily on Redis for distributed caching, session persistence, pub/sub messaging, and real-time streams. However, developers and AI coding agents encounter significant friction when using standard Redis tools:
- The native `redis-cli` relies on low-level commands with steep cognitive overhead and lacks multi-instance context switching.
- Operations engineers juggle local development containers, remote staging, and cloud production instances, requiring repeated credential entry.
- Destructive commands and un-pipelined `KEYS *` queries risk server lockups and cascading failures on production instances.
- Standard CLI outputs lack color, structured tables, and pagination for humans, while dumping noisy text payloads that waste LLM context tokens for AI agents.

This specification defines a modern, dual-mode CLI binary named **`redis`**, providing:
1. **Multi-Context Engine Switching**: Docker/Kubernetes-style context commands (`redis context ls`, `redis context use`, `redis context create`).
2. **Safe, Intuitive Key Exploration**: Paginated, non-blocking `SCAN` table views (`redis show --page=1 --limit=50 --format=table`).
3. **Dual DX Support**: Rich ANSI tables and an interactive TUI (`redis explore`) for humans, paired with compact JSON, field filtering, and semantic exit codes for AI agents and CI pipelines.
4. **Resilient Safety Guardrails**: Mandatory `--force` for non-TTY mass operations, dry-run simulation, and a 256KB memory preview ceiling.

---

## 2. High-Level Architecture & Component Boundaries

The system is designed with strict modular isolation into 6 core packages:

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           CLI Entrypoint (main.go)                      │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
           ┌─────────────────────────┴─────────────────────────┐
           ▼                                                   ▼
┌──────────────────────┐                             ┌──────────────────────┐
│  cmd/ (Cobra Tree)   │                             │  pkg/config/         │
│  - context (ls/use)  │                             │  - Context Store     │
│  - show / get / set  │◄────────────────────────────┤  - Resolver (3-tier) │
│  - summary / inspect │                             │  - Env Interpolation │
│  - explore (TUI)     │                             └──────────────────────┘
└──────────┬───────────┘
           │
           ├─────────────────────────┬─────────────────────────┐
           ▼                         ▼                         ▼
┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐
│  pkg/client/         │  │  pkg/format/         │  │  pkg/safety/         │
│  - Connection Pool   │  │  - Lipgloss Tables   │  │  - Guardrails        │
│  - SCAN & Pipelines  │  │  - JSON Compact/Nd   │  │  - IsTTY Check       │
│  - Metrics Extractor │  │  - Value Decoder     │  │  - Memory Cap 256KB  │
│  - TLS Handshake     │  │  - TUI Explorer      │  │  - Dry-Run Counter   │
└──────────┬───────────┘  └──────────────────────┘  └──────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  Target Redis Engines (RESP3 / TCP / TLS)               │
│        (Local Docker :6379, Realtime :6380, Upstash, Cloud Clusters)    │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities:
1. **`cmd/`**: Defines the CLI command tree, flags, and help text using GitHub/Cobra.
2. **`pkg/config/`**: Manages the YAML configuration at `~/.config/redis/contexts.yaml`, handles POSIX file permissions (`0600`), sanitizes exported configs, and evaluates context precedence (`--context` > `REDIS_CONTEXT` > `current-context`).
3. **`pkg/client/`**: Wraps `go-redis/v9` with connection pooling, health checks, automatic TLS certificate loading, non-blocking `SCAN` iteration, and metadata pipelining.
4. **`pkg/format/`**: Formats output streams based on target media. Renders TrueColor tables for TTYs, compact single-line JSON for agent pipes, and mounts the Bubble Tea TUI.
5. **`pkg/safety/`**: Intercepts destructive actions, verifies TTY presence, evaluates `--dry-run` and `--force` flags, and caps value reads at 256KB.

---

## 3. Configuration & Context Store Schema

The context store resides at `$HOME/.config/redis/contexts.yaml`.

```yaml
version: 1
current-context: local-common
contexts:
  - name: local-common
    host: 127.0.0.1
    port: 6379
    db: 0
    auth:
      username: ""
      password: "${LOCAL_REDIS_PASSWORD}"
    tls:
      enabled: false
    timeout_ms: 2000

  - name: local-realtime
    host: 127.0.0.1
    port: 6380
    db: 0
    auth:
      password: ""
    tls:
      enabled: false
    timeout_ms: 2000

  - name: staging-cloud
    host: redis-staging.internal.net
    port: 6379
    db: 1
    auth:
      username: app-user
      password: "${STAGING_REDIS_SECRET}"
    tls:
      enabled: true
      insecure_skip_verify: false
      ca_cert_file: "/etc/ssl/certs/internal-ca.pem"
    timeout_ms: 3000
```
**Description:** Canonical schema for multi-context Redis configuration with environment variable secret interpolation and TLS support.

---

## 4. Command Hierarchy & Subcommand Signatures

```text
redis
├── context
│   ├── ls                    # List available contexts with active marker (*)
│   ├── use <name>            # Switch global active context
│   ├── create <name> [flags] # Create new context profile
│   ├── delete <name>         # Remove a context profile
│   ├── current               # Print active context name
│   ├── export [--include-secrets] # Export sanitized YAML configuration
│   └── import <file>         # Import context profiles from YAML
├── show [flags]              # Paginated key browser (safe SCAN + table view)
├── inspect <key> [flags]     # Detailed key inspector (Type, TTL, Memory, Value)
├── get <key>                 # Direct value fetch (with 256KB preview cap)
├── set <key> <val> [flags]   # Set value with optional TTL (--ttl=300s)
├── del <key...> [flags]      # Safe delete (supports --pattern, --dry-run, --force)
├── summary [flags]           # Compact system health snapshot (< 500 bytes JSON)
└── explore                   # Launch full-screen interactive Bubble Tea TUI
```

### Global Persistent Flags:
- `--context <name>`: Override active context for this single invocation.
- `--format <table|json|yaml|plain>`: Output formatter (default: `table` for TTY, `json` for pipe).
- `--json`: Shorthand for `--format=json --compact`.
- `--compact`: Output single-line JSON without indentation.
- `--fields <list>`: Comma-separated list of fields to include in JSON output.
- `--dry-run`: Simulate write/delete commands without altering Redis state.
- `--force`: Bypass interactive confirmation on destructive actions.
- `--no-color`: Disable ANSI color formatting.
- `--timeout <ms>`: Socket connection and execution timeout in milliseconds.

---

## 5. Execution Flow & Algorithmic Mechanics

### 5.1 Safe Key Browsing (`redis show`)
```text
1. Parse flags (--page=1, --limit=50, --pattern="user:*").
2. Calculate target cursor offset or stream until limit is satisfied.
3. Execute: SCAN cursor MATCH <pattern> COUNT <limit>.
4. Open Pipeline:
   - For each returned key, queue TYPE <key>, TTL <key>, MEMORY USAGE <key>.
5. Execute Pipeline and assemble KeyMetadata structs.
6. If output is TTY:
     Render Lipgloss table with color badges (Green=String, Purple=Hash, Blue=Set).
   Else:
     Emit compact JSON array filtered by --fields.
```

### 5.2 Destructive Action Interceptor (`redis del` / `redis flushdb`)
```text
1. Identify if target operation is destructive.
2. If --dry-run is present:
     Count matching keys, calculate memory footprint, return JSON summary, exit 0.
3. Inspect environment via isatty.IsTerminal(os.Stdin.Fd()):
   - If TTY and --force NOT present:
       Prompt: "⚠️ About to delete N keys on context [prod]. Confirm? (y/N): "
   - If Non-TTY (Agent/CI) and --force NOT present:
       Emit JSON to Stderr: {"error": "destructive action requires --force", "code": 4}
       Exit with code 4 (SAFETY_GUARDRAIL_VIOLATION).
4. Execute atomic deletion batch via UNLINK / DEL pipeline.
```

---

## 6. Dual-Mode DX: Human vs AI Agent Specification

| Feature | Human Developer DX | AI Agent / Script DX |
| :--- | :--- | :--- |
| **Output Channel** | TTY (Standard terminal) | Non-TTY / Pipe / Subprocess |
| **Visual Presentation** | 24-bit TrueColor tables, badges, icons | Compact 1-line JSON / NDJSON |
| **Field Filtering** | Full visual column layout | Surgical `--fields=key,type,ttl` |
| **Destructive Prompting** | Interactive Y/N prompt with warning | Immediate exit code 4 unless `--force` |
| **Memory Ceiling** | 256KB truncated preview + notice | Exact byte count + truncated flag |
| **System Diagnostics** | Human-readable `redis summary` cards | `< 500` byte compact JSON snapshot |
| **Error Handling** | Natural language error message | Structured JSON on Stderr + Semantic Exit Code |

### Standard Semantic Exit Codes:
- `0`: `EXIT_OK` — Command executed successfully.
- `1`: `EXIT_CONNECTION_ERROR` — TCP socket failure, timeout, or DNS resolution error.
- `2`: `EXIT_AUTH_ERROR` — Invalid password, username, or ACL restriction.
- `3`: `EXIT_KEY_NOT_FOUND` — Key does not exist.
- `4`: `EXIT_SAFETY_VIOLATION` — Missing `--force` flag on non-interactive destructive action.
- `5`: `EXIT_SYNTAX_ERROR` — Invalid flags or unknown command.

---

## 7. Verification & Empirical Proof Plan

### Phase 1: Unit & Component Tests
- `pkg/config`: Verify YAML serialization, 0600 permission enforcement, and environment variable expansion (`${PASS}`).
- `pkg/safety`: Verify `isatty` mock tests, `--dry-run` counting, and exit code 4 on unforced deletion.
- `pkg/format`: Verify JSON compact serialization and Lipgloss table rendering.

### Phase 2: Live Integration Tests
Target the existing local Docker Redis containers:
- `capstone-redis-common` on `127.0.0.1:6379`
- `capstone-redis-realtime` on `127.0.0.1:6380`

1. Context Creation & Switching:
   ```bash
   ./redis context create local-common --host 127.0.0.1 --port 6379
   ./redis context create local-realtime --host 127.0.0.1 --port 6380
   ./redis context use local-common
   ```
2. Safe Key Exploration:
   ```bash
   ./redis show --page=1 --limit=10 --format=table
   ./redis show --json --fields=key,type,ttl | jq .
   ```
3. Agent Summary Snapshot:
   ```bash
   ./redis summary --json | jq .
   ```
4. Safety Guardrail Verification:
   ```bash
   # Must fail with exit code 4 without --force
   ./redis del "test:*" </dev/null && echo "FAILED" || echo "PASSED: Exit code 4"
   ```

---

## 8. Implementation Roadmap & Plan Handoff

The implementation can be executed through either:
1. **Micro Tactical TDD (`/spp:plan`)**: Bite-sized 2–5 minute steps, writing failing test -> implementing code -> passing test -> git commit.
2. **Macro Architecture Plan (`/james:plan`)**: Multi-phase roadmap covering project scaffolding, context engine, client adapters, safety gates, and TUI exploration with an adversarial red-team review.
