package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"redis-cli-tool/pkg/safety"
)

func TestAgentDX_Summary(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Create test context
	ResetFlags()
	rootCmd.SetArgs([]string{"context", "create", "test-summary-ctx", "--host", "127.0.0.1", "--port", "6379"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	// 1. Run summary --json
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"summary", "--json", "--compact"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run summary: %v", err)
	}

	jsonBytes := bytes.TrimSpace(buf.Bytes())
	if len(jsonBytes) >= 500 {
		t.Fatalf("expected summary JSON payload < 500 bytes, got %d bytes: %s", len(jsonBytes), string(jsonBytes))
	}

	var summary ServerSummary
	if err := json.Unmarshal(jsonBytes, &summary); err != nil {
		t.Fatalf("failed to unmarshal summary JSON: %v", err)
	}
	if summary.Context != "test-summary-ctx" {
		t.Fatalf("expected context 'test-summary-ctx', got %q", summary.Context)
	}
}

func TestAgentDX_FieldFilteringAndCompact(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Create context and seed key
	ResetFlags()
	rootCmd.SetArgs([]string{"context", "create", "test-filter-ctx", "--host", "127.0.0.1", "--port", "6379"})
	_ = rootCmd.Execute()

	ResetFlags()
	rootCmd.SetArgs([]string{"set", "test:agent:key1", "sample_val"})
	_ = rootCmd.Execute()

	// Run show with --json --compact --fields=key,type
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"show", "--pattern", "test:agent:*", "--json", "--compact", "--fields=key,type"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to execute show with fields: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if strings.Contains(output, "\n") {
		t.Fatalf("expected compact 1-line output, got multiline: %s", output)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("failed to parse filtered JSON: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected at least 1 item returned")
	}

	item := items[0]
	if _, hasKey := item["key"]; !hasKey {
		t.Error("expected 'key' in filtered output")
	}
	if _, hasType := item["type"]; !hasType {
		t.Error("expected 'type' in filtered output")
	}
	if _, hasMemory := item["memory_bytes"]; hasMemory {
		t.Error("memory_bytes should have been stripped by field filter")
	}

	// Clean up
	ResetFlags()
	rootCmd.SetArgs([]string{"del", "test:agent:key1", "--force"})
	_ = rootCmd.Execute()
}

func TestAgentDX_DelGuardrails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Create context and seed key
	ResetFlags()
	rootCmd.SetArgs([]string{"context", "create", "test-del-ctx", "--host", "127.0.0.1", "--port", "6379"})
	_ = rootCmd.Execute()

	ResetFlags()
	rootCmd.SetArgs([]string{"set", "test:del:guarded", "val"})
	_ = rootCmd.Execute()

	// 1. Simulated non-TTY pattern del without force should return SafetyError
	ResetFlags()
	buf.Reset()
	// Set stdin to empty reader to simulate non-TTY pipe /dev/null
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"del", "test:del:*"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected safety error for pattern deletion without force")
	}
	safetyErr, ok := err.(*safety.SafetyError)
	if !ok || safetyErr.Code != safety.ExitSafetyViolation {
		t.Fatalf("expected SafetyError code 4, got: %v", err)
	}

	// 2. Dry run with pattern
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"del", "test:del:*", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run dry-run: %v", err)
	}
	if !strings.Contains(buf.String(), "[DRY-RUN]") || !strings.Contains(buf.String(), "0 keys deleted") {
		t.Fatalf("unexpected dry-run output: %s", buf.String())
	}

	// 3. Del with --force
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"del", "test:del:*", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to delete with --force: %v", err)
	}
	if !strings.Contains(buf.String(), "Successfully unlinked") {
		t.Fatalf("expected successful unlink message, got: %s", buf.String())
	}
}
