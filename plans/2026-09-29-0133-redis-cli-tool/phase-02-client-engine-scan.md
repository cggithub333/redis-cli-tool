---
phase: 2
title: "Client Engine & Non-Blocking SCAN"
status: completed
priority: P1
effort: "2h"
dependencies: ["phase-01-core-scaffolding-config.md"]
---

# Phase 02: Client Engine & Non-Blocking SCAN

## Overview
Builds the client adapter wrapping `github.com/redis/go-redis/v9`. Embeds `*redis.Client` to expose standard Redis operations to downstream subcommands, configures connection pooling, handles TLS handshake configuration, provides health verification (Ping), and implements non-blocking `SCAN` iteration with safely chunked metadata pipelines (max 200 keys per pipeline flush).

## Requirements
- Functional:
  - Create `Client` struct embedding `*redis.Client` initialized from resolved `Context` configuration.
  - Parameterize constructor as `NewClient(cfg *config.Context) (*Client, error)` without variable shadowing.
  - Support TLS/mTLS (CA certificate, client cert, private key, insecure skip verify).
  - Implement `Ping(ctx context.Context)` with configurable connection timeout and latency sampling.
  - Implement `ScanKeys(ctx context.Context, pattern string, cursor uint64, count int64)` returning paginated keys.
  - Chunk pipelined metadata queries into safe batches (max 200 keys per pipeline execution) to avoid blocking Redis production threads or exhausting CLI memory.
- Non-functional:
  - Zero use of blocking `KEYS *` queries.
  - Pipelined query execution latency under 10ms for batches of 200 keys.

## Architecture
```text
Context Config (Host, Port, DB, TLS)
               │
               ▼
pkg/client/client.go:
  - Setup redis.Options (Addr, DB, Password, TLSConfig)
  - Return *Client { *redis.Client, cfg }
               │
               ▼
pkg/client/scanner.go:
  1. SCAN cursor MATCH <pattern> COUNT <limit>
  2. For keys in chunks of 200:
     - redis.Pipelined:
       - Pipe.Type(ctx, key)
       - Pipe.TTL(ctx, key)
       - Pipe.MemoryUsage(ctx, key)
  3. Assemble & return []KeyMetadata
```

## Related Code Files
- Create: `pkg/client/client.go`
- Create: `pkg/client/scanner.go`
- Create: `pkg/client/types.go`
- Create: `pkg/client/client_test.go`
- Create: `pkg/client/scanner_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Implement Client Factory & Connection Pooling
- **Files**: `pkg/client/client.go`, `pkg/client/types.go`, `pkg/client/client_test.go`
- **Action**: Implement `NewClient(cfg *config.Context) (*Client, error)` embedding `*redis.Client`. Support standalone connection, DB selection, custom dial timeouts, and optional TLS configuration.
- **Verification Command**: `go test -v -run TestClientConnection ./pkg/client/...`
- **Expected Output**: PASS testing connection against localhost:6379 (`capstone-redis-common`).

### Step 2: Implement SCAN Cursor Iterator & Chunked Pipeline Metadata Fetcher
- **Files**: `pkg/client/scanner.go`, `pkg/client/scanner_test.go`
- **Action**: Implement `ScanKeys(ctx context.Context, pattern string, cursor uint64, count int64) (*ScanResult, error)`. Chunk pipelined commands in batches of 200 keys. Gracefully handle errors on `MEMORY USAGE` (fallback to 0).
- **Verification Command**: `go test -v -run TestScanKeysPipeline ./pkg/client/...`
- **Expected Output**: PASS verifying keys, types, and TTL returned in chunked batches.

### Step 3: Implement Live Integration Ping & Health Check
- **Files**: `pkg/client/client.go`, `pkg/client/client_test.go`
- **Action**: Implement `Ping(ctx context.Context) error` and latency calculation.
- **Verification Command**: `go test -v -run TestPing ./pkg/client/...`
- **Expected Output**: PASS verifying latency calculation.

## Success Criteria
- [x] `Client` embeds `*redis.Client` allowing all downstream commands (`Get`, `Set`, `Info`, `Del`) to execute without wrapper boilerplate.
- [x] Connects cleanly to live Docker Redis instances (`localhost:6379` and `localhost:6380`).
- [x] `ScanKeys` executes without errors and chunks pipelines safely without memory bloat.

## Risk Assessment
- **Risk**: Enormous SCAN iterations blocking CLI memory.
- **Mitigation**: 200-key pipeline chunk ceiling prevents massive memory buffers.
