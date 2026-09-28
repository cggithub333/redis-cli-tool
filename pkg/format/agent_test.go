package format

import (
	"strings"
	"testing"
)

func TestFilterFields(t *testing.T) {
	input := map[string]interface{}{
		"key":          "test:user:1",
		"type":         "string",
		"ttl":          3600,
		"memory_bytes": 1024,
	}

	filtered := FilterFields(input, []string{"key", "type"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(filtered))
	}
	if filtered["key"] != "test:user:1" || filtered["type"] != "string" {
		t.Fatalf("unexpected filtered content: %v", filtered)
	}
	if _, exists := filtered["memory_bytes"]; exists {
		t.Fatal("memory_bytes should have been stripped")
	}
}

func TestSerializeJSON(t *testing.T) {
	input := map[string]string{"foo": "bar"}

	// Compact mode (single line, no newlines)
	compact, err := SerializeJSON(input, true)
	if err != nil {
		t.Fatalf("failed to serialize compact: %v", err)
	}
	if strings.Contains(compact, "\n") {
		t.Fatalf("compact JSON should not contain newlines: %s", compact)
	}

	// Indented mode
	indented, err := SerializeJSON(input, false)
	if err != nil {
		t.Fatalf("failed to serialize indented: %v", err)
	}
	if !strings.Contains(indented, "\n") {
		t.Fatalf("indented JSON should contain newlines: %s", indented)
	}
}
