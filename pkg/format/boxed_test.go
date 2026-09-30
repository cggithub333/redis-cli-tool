package format

import (
	"os"
	"strings"
	"testing"
)

func TestDetectTerminalWidth(t *testing.T) {
	// Test with FZF_PREVIEW_COLUMNS
	os.Setenv("FZF_PREVIEW_COLUMNS", "110")
	defer os.Unsetenv("FZF_PREVIEW_COLUMNS")
	if w := DetectTerminalWidth(); w != 110 {
		t.Fatalf("expected 110 from FZF_PREVIEW_COLUMNS, got %d", w)
	}

	// Test with COLUMNS
	os.Unsetenv("FZF_PREVIEW_COLUMNS")
	os.Setenv("COLUMNS", "95")
	defer os.Unsetenv("COLUMNS")
	if w := DetectTerminalWidth(); w != 95 {
		t.Fatalf("expected 95 from COLUMNS, got %d", w)
	}
}

func TestDetectStringLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, "json"},
		{`[1, 2, 3]`, "json"},
		{"---\nname: myapp\nversion: 1.0", "yaml"},
		{"server:\n  port: 8080\n  host: 127.0.0.1", "yaml"},
		{"<root><child>value</child></root>", "xml"},
		{"SELECT id, name FROM users WHERE active = 1;", "sql"},
		{"Notice: Scheduled database maintenance window on Sunday.", "text"},
		{"Just a normal plain string.", "text"},
	}

	for _, tc := range tests {
		lang := DetectStringLanguage(tc.input)
		if lang != tc.expected {
			t.Errorf("for input %q, expected %q, got %q", tc.input, tc.expected, lang)
		}
	}
}

func TestRenderBoxedContent(t *testing.T) {
	sampleJSON := `{"id": 1001, "name": "Alex", "active": true}`
	boxed := RenderBoxedContent(sampleJSON, "json", 80)

	// Verify background color code is present
	if !strings.Contains(boxed, "\x1b[48;2;38;38;38m") {
		t.Fatal("expected dark gray background escape code in boxed output")
	}

	// Verify content lines exist
	if !strings.Contains(boxed, "Alex") {
		t.Fatal("expected content inside boxed output")
	}

	// Test long line wrapping
	longText := strings.Repeat("VeryLongWordWithoutSpaces", 10)
	wrappedBox := RenderBoxedContent(longText, "text", 50)
	if !strings.Contains(wrappedBox, "\x1b[48;2;38;38;38m") {
		t.Fatal("expected background in wrapped box")
	}
}

func TestRenderInspectContent(t *testing.T) {
	jsonResult := &DecodedResult{
		ValueType: TypeJSON,
		Formatted: `{"status": "ok"}`,
	}

	// Compact mode test
	compactOut := RenderInspectContent(jsonResult, 80, true)
	if compactOut != `{"status": "ok"}` {
		t.Fatalf("expected raw formatted in compact mode, got %q", compactOut)
	}

	// Boxed mode test
	boxedOut := RenderInspectContent(jsonResult, 80, false)
	if !strings.Contains(boxedOut, "\x1b[48;2;38;38;38m") {
		t.Fatal("expected boxed styling for JSON")
	}

	// Collections test
	hashResult := &DecodedResult{
		ValueType: TypeHash,
		Formatted: "┌───────┬───────┐\n│ FIELD │ VALUE │\n└───────┴───────┘",
	}
	hashOut := RenderInspectContent(hashResult, 80, false)
	if hashOut != hashResult.Formatted {
		t.Fatalf("expected collections to retain table formatting, got %q", hashOut)
	}
}
