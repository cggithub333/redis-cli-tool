---
name: redis-cli
description: Modern Redis CLI with Docker-style multi-context management, non-blocking SCAN iteration, syntax-highlighted value inspection, bounded collection preview, and agent DX (--json --compact, --fields). Use when inspecting, querying, caching, debugging, or managing Redis keys and health across local Docker engines and cloud providers (Layerbase, Upstash, AWS, Redis Official).
---

# Redis CLI Tool (`redis`)

A high-performance, single-binary CLI tool for managing, browsing, and inspecting Redis instances across multiple connection contexts (local Docker containers, Layerbase, Upstash, AWS ElastiCache, and Redis Official Cloud).

Designed specifically for both human developers and autonomous AI coding agents.

---

## ⚡ Agent Quick Start & Golden Rules

When invoking `redis` within automated agent workflows or scripts, adhere to these golden rules:

1. **Always Use Compact JSON in Scripts**:
   Append `--json --compact` to guarantee machine-parseable single-line output and minimize token consumption in agent context budgets.
   ```bash
   redis summary --json --compact
   redis show --pattern "user:*" --json --compact --fields=key,type,ttl
   ```
2. **Never Run `KEYS *`**:
   The `redis show` command automatically employs non-blocking pipelined `SCAN` (chunk size 50) and never blocks Redis event loops.
3. **Handle Semantic Exit Codes**:
   - `0`: Success
   - `1`: General Error (e.g. invalid arguments)
   - `2`: Connection Failure (e.g. host unreachable, auth failure)
   - `3`: Key Not Found
   - `4`: Safety Guardrail Blocked (e.g. pattern delete without `--force` in non-interactive environment)
   - `5`: TTY Required (e.g. attempting to run `redis explore` TUI in non-interactive agent shell)
4. **Destructive Operations Require `--force`**:
   When deleting keys via glob patterns (`*`) or running in non-TTY agent shells, always pass `--force` (or `--dry-run` to preview).
   ```bash
   redis del "cache:temp:*" --force
   ```
5. **Context Precedence Hierarchy**:
   1. Explicit CLI Flag: `--context <name>`
   2. Environment Variable: `REDIS_CONTEXT=<name>`
   3. Active Context Configuration: `current-context` in `~/.config/redis/contexts.yaml`

---

## 🧭 Multi-Context Management

The CLI supports Docker/Kubernetes-style context switching, allowing seamless hopping between local containers and cloud Redis engines.

### 1. List All Contexts
Displays active context, supplier/provider, endpoint, database index, health status, and warm round-trip latency.
```bash
# Terminal table view:
redis context ls

# Machine-parseable JSON:
redis context ls --format json
```

### 2. Switch Active Context
```bash
redis context use capstone-redis-001
```

### 3. Show Current Context Name
```bash
redis context current
```

### 4. Create a Context
Supports both cloud connection URIs (`redis://` and `rediss://`) and individual connection flags:

```bash
# Using Cloud URI (TLS automatically detected from rediss://):
redis context create capstone-cloud --uri "rediss://default:token@capstone-redis.layerbase.dev:6379/0"

# Using Individual Flags (Local Docker Engine):
redis context create local-dev --host 127.0.0.1 --port 6379 --supplier "Local container"

# Overriding URI parameters:
redis context create cloud-staging --uri "rediss://...upstash.io:6379" --db 2
```

### 5. Rename an Existing Context
Renames a profile and automatically updates `current-context` if active:
```bash
redis context rename old-name new-name
# Alias:
redis context mv old-name new-name

# Optional supplier update:
redis context rename dev-cluster prod-cluster --supplier "Redis Official"
```

### 6. Export & Import Configurations
Config file permissions are strictly enforced to POSIX `0600` (`~/.config/redis/contexts.yaml`).
```bash
# Export with passwords masked to ${ENV_VAR} placeholders:
redis context export > contexts.yaml

# Export with plain secrets (requires explicit flag):
redis context export --include-secrets > backup.yaml

# Import contexts from file or stdin:
redis context import contexts.yaml
cat contexts.yaml | redis context import -
```

---

## 🔍 Data Inspection & Key Browsing

### 1. Health Snapshot (`redis summary`)
Outputs server version, role, memory consumption, key counts, and hit rates:
```bash
# Human card view:
redis summary

# Agent DX (< 500 bytes token-budget JSON):
redis summary --json --compact
```
*Output sample:*
```json
{"context":"capstone-redis-001","version":"7.2.15","role":"master","uptime_sec":932,"uptime":"15m32s","clients":1,"memory":"1.04M","keys":14,"hit_rate_pct":98.2,"ops_per_sec":4}
```

### 2. Key Browser (`redis show`)
Scans keys non-blockingly with pagination and metadata:
```bash
# Browse first 20 keys matching pattern:
redis show --pattern "session:*" --limit 20

# Next page via cursor:
redis show --pattern "session:*" --cursor 1024 --limit 20

# Agent compact field projection:
redis show --pattern "api:*" --json --compact --fields=key,type,ttl
```

### 3. Key Inspector (`redis inspect`)
Inspects data type, memory footprint, TTL, encoding, and decoded value:
- **JSON**: Formatted with syntax highlighting.
- **Hashes, Sets, Lists, ZSets, Streams**: Rendered in structured tables with bounded collection limits (max 100 items).
- **Binary**: Clean hexadecimal dump.
- **Safety Ceiling**: Strings capped at 256KB preview to avoid client OOM (use `--full` to bypass).

```bash
redis inspect user:1001:profile
redis inspect session:queue --json
```

### 4. Basic Get / Set
```bash
# Set with TTL:
redis set cache:auth:token "xyz123" --ttl 300s

# Get value:
redis get cache:auth:token
redis get user:profile --full
```

---

## 🗑️ Destructive Operations & Guardrails

### Safe Delete (`redis del`)
Deletes keys using non-blocking chunked `UNLINK` (500 keys per batch) to avoid freezing Redis servers.

```bash
# Delete single key:
redis del cache:auth:token

# Dry-run pattern match (simulates without deleting):
redis del "temp:*" --dry-run

# Delete pattern with mandatory --force:
redis del "temp:*" --force
```

---

## 💻 Common Agent Coding Workflows

### Scenario A: Verifying Cache Invalidation in Unit/E2E Tests
```bash
# 1. Ensure test key exists before test action:
redis get test:user:cache --json

# 2. Trigger application cache invalidation...

# 3. Verify key was deleted:
redis get test:user:cache
# Expects exit code 3 (ExitKeyNotFound)
```

### Scenario B: Automated Context Provisioning in CI/CD
```bash
# Setup staging context from pipeline environment variable:
redis context create ci-env --uri "$REDIS_STAGING_URL" --force
redis context use ci-env
redis summary --json --compact
```

### Scenario C: Diagnosing Memory Bloat in Production
```bash
# 1. Quick health check:
redis summary --json --compact

# 2. Find largest keys in a domain:
redis show --pattern "events:*" --json --compact --fields=key,type,memory_bytes
```
