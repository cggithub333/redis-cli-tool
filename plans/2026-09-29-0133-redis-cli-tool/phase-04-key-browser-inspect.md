---
phase: 4
title: "Key Exploration & Adaptive Decoder"
status: completed
priority: P1
effort: "2.5h"
dependencies: ["phase-02-client-engine-scan.md", "phase-03-context-commands.md"]
---

# Phase 04: Key Exploration & Adaptive Decoder

## Overview
Implements key exploration and value inspection commands (`redis show`, `redis inspect`, `redis get`, `redis set`). Builds an adaptive value decoder that automatically detects JSON via `json.Valid()`, syntax-highlights formatting with Lipgloss, renders binary data as a formatted 16-column hex dump, reads collections safely via bounded iterators (`HSCAN`, `SSCAN`, `LRANGE 0 100` capped at 100 items), and enforces a 256KB memory preview safety ceiling.

## Requirements
- Functional:
  - `redis show`: Browse keys using `SCAN` with flags `--pattern`, `--cursor <uint64>` (default 0), `--page <int>` (client-side skip), `--limit <int>`, `--format=table/json`.
  - `redis inspect <key>`: Display comprehensive key card (Key, Type, TTL, Memory, Encoding, Fields count, Value preview).
  - Safe Collection Previews: Never execute raw `HGETALL` or `SMEMBERS` on unknown keys; use bounded iterators (`HSCAN`, `SSCAN`, `LRANGE 0 100`) capped at max 100 elements.
  - `redis get <key>`: Retrieve string value with 256KB preview cap unless `--full` is specified.
  - `redis set <key> <value> [--ttl=300s]`: Store key with optional TTL.
  - Adaptive Decoder: Use `json.Valid()` to detect JSON and format with indentation and Lipgloss coloring. Detect non-printable bytes and format as 16-column hex dump.
- Non-functional:
  - Memory consumption capped at < 50MB even when inspecting large collection keys.

## Architecture
```text
redis show / inspect
        │
        ▼
pkg/client/scanner.go (Fetch key metadata via SCAN + pipeline)
        │
        ▼
pkg/format/decoder.go:
  - json.Valid(bytes)  ──► json.Indent() + Lipgloss token styling
  - IsBinary(bytes)    ──► hex.Dump() formatted view
  - IsHash/List/Set    ──► HSCAN / LRANGE 0 100 (Hard element cap: 100 items)
  - Size > 256KB       ──► Truncate with "[Truncated: showing 256KB of 12MB. Use --full to stream]"
```

## Related Code Files
- Create: `cmd/show.go`
- Create: `cmd/inspect.go`
- Create: `cmd/get_set.go`
- Create: `pkg/format/decoder.go`
- Create: `pkg/format/decoder_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Implement Adaptive Value Decoder & Bounded Collection Reader
- **Files**: `pkg/format/decoder.go`, `pkg/format/decoder_test.go`
- **Action**: Implement `DecodeValue(client *client.Client, key, keyType string) (*DecodedResult, error)` using `json.Valid()`, Lipgloss color styling, 16-col hex dump, and bounded collection sampling (`HSCAN`/`LRANGE` capped at 100 items).
- **Verification Command**: `go test -v -run TestAdaptiveDecoder ./pkg/format/...`
- **Expected Output**: PASS testing JSON, string, collection, and binary payloads.

### Step 2: Implement Key Metadata Table Formatting
- **Files**: `cmd/show.go`, `cmd/inspect.go`
- **Action**: Connect `pkg/format/table.go` (from Phase 03) to render keys list with Type badges, TTL countdowns, and Memory bytes.
- **Verification Command**: `go vet ./cmd/...`
- **Expected Output**: Exit code 0.

### Step 3: Implement `redis show`, `redis inspect`, `redis get`, `redis set` with Fixture Seeding
- **Files**: `cmd/show.go`, `cmd/inspect.go`, `cmd/get_set.go`
- **Action**: Wire commands with Cobra. Seed sample test keys via Docker Redis before running show verification:
  ```bash
  docker exec capstone-redis-common redis-cli MSET test:user:1 "Alice" test:user:2 "Bob" test:config '{"env":"dev","debug":true}'
  ```
- **Verification Command**: `go run main.go show --limit=5`
- **Expected Output**: Displays colorful table of keys from active Redis instance.

## Success Criteria
- [x] `redis show --format=table` renders key name, type, TTL, and memory usage against seeded test keys.
- [x] `redis inspect <key>` on a JSON key displays pretty-printed syntax highlighted output.
- [x] Large collections use bounded iterators capped at 100 items without memory bloat.

## Risk Assessment
- **Risk**: Massive hash with 5 million fields crashing CLI during inspect.
- **Mitigation**: Bounded iterator with 100-item ceiling strictly avoids `HGETALL` OOM.
