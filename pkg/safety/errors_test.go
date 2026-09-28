package safety

import (
	"bytes"
	"strings"
	"testing"
)

func TestSemanticErrors(t *testing.T) {
	err := NewSafetyError(ExitSafetyViolation, "destructive action requires --force flag")
	if err.Code != 4 {
		t.Fatalf("expected code 4, got %d", err.Code)
	}

	buf := new(bytes.Buffer)
	if err := err.EmitJSON(buf); err != nil {
		t.Fatalf("failed to emit JSON error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"code":4`) || !strings.Contains(out, `"error":"destructive action requires --force flag"`) {
		t.Fatalf("unexpected JSON error output: %s", out)
	}
}
