package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdaptiveDecoder_JSON(t *testing.T) {
	rawJSON := []byte(`{"id":123,"name":"Alice","active":true}`)
	res := DecodeStringPayload(rawJSON, false)

	if res.ValueType != TypeJSON {
		t.Fatalf("expected TypeJSON, got %s", res.ValueType)
	}
	if !strings.Contains(res.Formatted, `"name": "Alice"`) {
		t.Fatalf("expected indented JSON, got: %s", res.Formatted)
	}
	if res.Truncated {
		t.Fatal("small payload should not be truncated")
	}
}

func TestAdaptiveDecoder_PlainText(t *testing.T) {
	rawText := []byte("Hello, Redis Multi-Context CLI!")
	res := DecodeStringPayload(rawText, false)

	if res.ValueType != TypeString {
		t.Fatalf("expected TypeString, got %s", res.ValueType)
	}
	if res.Formatted != string(rawText) {
		t.Fatalf("expected %q, got %q", string(rawText), res.Formatted)
	}
}

func TestAdaptiveDecoder_Binary(t *testing.T) {
	rawBinary := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0xAA}
	res := DecodeStringPayload(rawBinary, false)

	if res.ValueType != TypeBinary {
		t.Fatalf("expected TypeBinary, got %s", res.ValueType)
	}
	if !strings.Contains(res.Formatted, "00000000") {
		t.Fatalf("expected hex dump offset in formatted output, got: %s", res.Formatted)
	}
}

func TestAdaptiveDecoder_Truncation(t *testing.T) {
	// Create payload larger than 256KB (e.g. 300KB)
	largeSize := MaxPreviewBytes + 50*1024
	largePayload := bytes.Repeat([]byte("A"), largeSize)

	// Without --full
	res := DecodeStringPayload(largePayload, false)
	if !res.Truncated {
		t.Fatal("expected truncated flag to be true")
	}
	if !strings.Contains(res.Formatted, "Truncated: showing 256KB") {
		t.Fatalf("expected truncation note, got: %s", res.Formatted[len(res.Formatted)-100:])
	}

	// With --full
	resFull := DecodeStringPayload(largePayload, true)
	if resFull.Truncated {
		t.Fatal("expected truncated flag to be false when full=true")
	}
	if strings.Contains(resFull.Formatted, "Truncated") {
		t.Fatal("did not expect truncation note when full=true")
	}
}

func TestIsBinaryData(t *testing.T) {
	if !IsBinaryData([]byte{0x00, 0x12}) {
		t.Error("expected true for null byte")
	}
	if IsBinaryData([]byte("Normal UTF-8 string with symbols! 🚀")) {
		t.Error("expected false for UTF-8 string")
	}
}

func TestAdaptiveDecoder_DisarmANSI(t *testing.T) {
	malicious := "User profile message: Welcome to production system \x1b[31;1mwarning text\x1b[0m completed successfully."
	res := DecodeStringPayload([]byte(malicious), false)
	if strings.Contains(res.Formatted, "\x1b") {
		t.Fatalf("expected ANSI escape code to be disarmed, got %q", res.Formatted)
	}
	if !strings.Contains(res.Formatted, "^[") {
		t.Fatalf("expected ^[ replacement, got %q", res.Formatted)
	}
}

func TestAdaptiveDecoder_RuneTruncation(t *testing.T) {
	// Vietnamese Unicode string with multi-byte runes
	str := "Xin chào thế giới Redis! Kiểm thử độ dài chuỗi UTF-8 an toàn."
	truncated := truncateRunes(str, 15)
	if !strings.HasSuffix(truncated, "...") {
		t.Fatalf("expected ellipsis suffix, got %s", truncated)
	}
	runes := []rune(truncated)
	if len(runes) > 15 {
		t.Fatalf("expected at most 15 runes, got %d", len(runes))
	}
}
