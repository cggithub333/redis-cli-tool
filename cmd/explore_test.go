package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"redis-cli-tool/pkg/tui"
)

func TestResizeCommand(t *testing.T) {
	sessionID := "test_cmd_resize_sess"
	_, cleanup := tui.InitResizeSession(sessionID)
	defer cleanup()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"__resize", sessionID, "toggle"})
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	if err != nil {
		t.Fatalf("expected __resize to succeed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "rebind(left,right)") {
		t.Fatalf("expected rebind in output, got: %s", out)
	}
}
