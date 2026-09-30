package tui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String(), err
}

func TestResizeState_InitAndCleanup(t *testing.T) {
	sessionID := "test_sess_001"
	path, cleanup := InitResizeSession(sessionID)
	defer cleanup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected state file to exist at %s: %v", path, err)
	}

	state := readResizeState(sessionID)
	if state.Size != 55 || state.LeaderMode {
		t.Fatalf("unexpected initial state: size=%d, leaderMode=%v", state.Size, state.LeaderMode)
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected state file to be removed after cleanup: %v", err)
	}
}

func TestResizeState_ToggleAndEsc(t *testing.T) {
	sessionID := "test_sess_002"
	_, cleanup := InitResizeSession(sessionID)
	defer cleanup()

	// 1. Toggle into Leader mode
	out, err := captureStdout(func() error {
		return HandleResize(sessionID, "toggle")
	})
	if err != nil {
		t.Fatalf("toggle failed: %v", err)
	}
	if !strings.Contains(out, "rebind(left,right)") || !strings.Contains(out, "change-header(") {
		t.Fatalf("unexpected toggle output: %s", out)
	}

	st := readResizeState(sessionID)
	if !st.LeaderMode {
		t.Fatal("expected leader mode to be true")
	}

	// 2. Esc while in leader mode -> exits leader mode, does not abort
	out, err = captureStdout(func() error {
		return HandleResize(sessionID, "esc")
	})
	if err != nil {
		t.Fatalf("esc in leader mode failed: %v", err)
	}
	if !strings.Contains(out, "unbind(left,right)") {
		t.Fatalf("expected unbind in esc output: %s", out)
	}

	st = readResizeState(sessionID)
	if st.LeaderMode {
		t.Fatal("expected leader mode to be false after esc")
	}

	// 3. Esc while NOT in leader mode -> aborts FZF
	out, err = captureStdout(func() error {
		return HandleResize(sessionID, "esc")
	})
	if err != nil {
		t.Fatalf("esc outside leader mode failed: %v", err)
	}
	if strings.TrimSpace(out) != "abort" {
		t.Fatalf("expected 'abort', got %q", out)
	}
}

func TestResizeState_LeftAndRight(t *testing.T) {
	sessionID := "test_sess_003"
	_, cleanup := InitResizeSession(sessionID)
	defer cleanup()

	// Initial size 55 (preview is 55%)
	// Right -> moves divider right (menu grows +5%, preview shrinks -5% -> 50%)
	out, err := captureStdout(func() error {
		return HandleResize(sessionID, "right")
	})
	if err != nil {
		t.Fatalf("right failed: %v", err)
	}
	if !strings.Contains(out, "change-preview-window(right:50%:wrap:border-rounded)") {
		t.Fatalf("unexpected right output: %s", out)
	}

	// Left 2x -> moves divider left (menu shrinks, preview grows: 50% -> 55% -> 60%)
	_, _ = captureStdout(func() error { return HandleResize(sessionID, "left") })
	out, err = captureStdout(func() error {
		return HandleResize(sessionID, "left")
	})
	if err != nil {
		t.Fatalf("left failed: %v", err)
	}
	if !strings.Contains(out, "change-preview-window(right:60%:wrap:border-rounded)") {
		t.Fatalf("unexpected left output: %s", out)
	}

	st := readResizeState(sessionID)
	if st.Size != 60 {
		t.Fatalf("expected size 60, got %d", st.Size)
	}
}
