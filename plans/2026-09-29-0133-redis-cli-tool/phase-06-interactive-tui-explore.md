---
phase: 6
title: "Interactive Full-Screen TUI"
status: pending
priority: P2
effort: "2.5h"
dependencies:
  - "phase-02-client-engine-scan.md"
  - "phase-03-context-commands.md"
  - "phase-04-key-browser-inspect.md"
  - "phase-05-agent-dx-safety-guardrails.md"
---

# Phase 06: Interactive Full-Screen TUI (`redis explore`)

## Overview
Builds an interactive full-screen Terminal User Interface (TUI) command `redis explore` (or `redis ui`) using the Charm.sh ecosystem (`github.com/charmbracelet/bubbletea` and `github.com/charmbracelet/bubbles`). Connects to context management (Phase 03) and safe deletion (Phase 05), providing real-time arrow/vim key navigation, live fuzzy search filtering, split-pane key inspection, and a strict non-TTY preflight rejection guardrail.

## Requirements
- Functional:
  - Command: `redis explore` (alias `redis ui`).
  - Left Pane: Scrollable list of keys fetched via `SCAN` with live search query input (`/`).
  - Right Pane: Live value preview, type badge, TTL countdown, and memory statistics.
  - Keyboard Controls:
    - `↑` / `k`, `↓` / `j`: Navigate keys.
    - `/`: Focus search input to filter by pattern.
    - `Enter`: Open modal / full-screen inspection of selected key.
    - `c`: Open interactive context switcher dialog (using Phase 03 context store).
    - `d`: Delete key with modal confirmation (using Phase 05 deletion logic).
    - `q` / `Esc`: Exit TUI.
  - Non-TTY Guardrail: When invoked without an interactive terminal (`!isatty`), exit immediately with error code 5 and message: `"redis explore requires an interactive TTY. Use 'redis show' instead."`
- Non-functional:
  - Smooth rendering without terminal screen flickering.

## Architecture
```text
redis explore
      │
      ▼
pkg/tui/guard.go:
  - if !isatty.IsTerminal(): return ErrNonTTYSession
      │
      ▼
Bubble Tea Event Loop (pkg/tui/model.go)
  ├── Update(msg tea.Msg)
  │     ├── tea.KeyMsg (Navigation, Filter, Shortcuts)
  │     └── tea.WindowSizeMsg (Dynamic resize)
  └── View() string
        ├── Left Pane: bubbles/table (Keys list)
        ├── Right Pane: Lipgloss styled value viewport
        └── Status Bar: Active context, key count, connection latency
```

## Related Code Files
- Create: `cmd/explore.go`
- Create: `pkg/tui/model.go`
- Create: `pkg/tui/views.go`
- Create: `pkg/tui/keys.go`
- Create: `pkg/tui/views_test.go`
- Create: `pkg/tui/model_test.go`

## Implementation Steps (Superpower Micro-Steps)

### Step 1: Scaffold Bubble Tea Model & Keybindings
- **Files**: `pkg/tui/model.go`, `pkg/tui/keys.go`, `pkg/tui/model_test.go`
- **Action**: Define `Model` struct holding keys list, active index, search input, and viewport.
- **Verification Command**: `go test -v -run TestTUIModel ./pkg/tui/...`
- **Expected Output**: PASS testing initial model state.

### Step 2: Implement Split-Pane Layout & Key Inspector View
- **Files**: `pkg/tui/views.go`, `pkg/tui/views_test.go`
- **Action**: Render left pane with key names and types; render right pane with syntax-highlighted value and TTL.
- **Verification Command**: `go test -v -run TestTUIViews ./pkg/tui/...`
- **Expected Output**: PASS testing view formatting.

### Step 3: Wire `redis explore` Command in Cobra & Verify Non-TTY Guardrail
- **Files**: `cmd/explore.go`
- **Action**: Create `exploreCmd` and launch Bubble Tea program (`tea.NewProgram(model, tea.WithAltScreen())`). Add non-TTY guardrail. Build binary and verify rejection when redirected from `/dev/null`.
- **Verification Command**: `go build -o bin/redis main.go && ./bin/redis explore </dev/null || [ $? -eq 5 ]`
- **Expected Output**: Exit code 5 with non-TTY error message.

## Success Criteria
- [ ] `redis explore` launches in alternate screen buffer without visual artifacts on TTY.
- [ ] Arrow keys and vim keys (`j`/`k`) navigate through keys smoothly.
- [ ] Non-TTY execution (`</dev/null`) exits cleanly with code 5 and does not crash or hang.

## Risk Assessment
- **Risk**: Terminal window too small causing panic on bounds calculations.
- **Mitigation**: Handle `tea.WindowSizeMsg` and render warning when width < 80 or height < 20.
