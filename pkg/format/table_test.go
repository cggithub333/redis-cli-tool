package format

import (
	"strings"
	"testing"
)

func TestTableRenderer(t *testing.T) {
	headers := []string{"NAME", "HOST", "PORT", "STATUS"}
	rows := [][]string{
		{"local-common", "127.0.0.1", "6379", StatusBadge("HEALTHY")},
		{"local-realtime", "127.0.0.1", "6380", StatusBadge("UNREACHABLE")},
	}

	rendered := RenderTable(headers, rows)
	if rendered == "" {
		t.Fatal("expected non-empty rendered table")
	}

	for _, h := range headers {
		if !strings.Contains(rendered, h) {
			t.Errorf("expected table to contain header %q", h)
		}
	}

	if !strings.Contains(rendered, "local-common") || !strings.Contains(rendered, "local-realtime") {
		t.Error("expected table to contain context names")
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"HEALTHY", "HEALTHY"},
		{"unreachable", "UNREACHABLE"},
		{"active", "ACTIVE"},
		{"unknown", "UNKNOWN"},
	}

	for _, tc := range tests {
		b := StatusBadge(tc.input)
		if !strings.Contains(b, tc.contains) {
			t.Errorf("StatusBadge(%q) = %q, expected to contain %q", tc.input, b, tc.contains)
		}
	}
}
