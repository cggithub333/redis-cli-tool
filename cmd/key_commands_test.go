package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestKeyCommands(t *testing.T) {
	// Set isolated test XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// 1. Create and switch to local test context
	ResetFlags()
	rootCmd.SetArgs([]string{"context", "create", "test-keys-ctx", "--host", "127.0.0.1", "--port", "6379"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	// 2. Test redis set
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"set", "test:key:string", "Hello Redis", "--ttl", "60s"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to execute set: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Fatalf("expected OK from set, got %q", buf.String())
	}

	// 3. Test redis set with JSON
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"set", "test:key:json", `{"env":"test","count":42}`, "--ttl", "60s"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to set JSON key: %v", err)
	}

	// 4. Test redis get
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"get", "test:key:string"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get string key: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Hello Redis" {
		t.Fatalf("expected 'Hello Redis', got %q", buf.String())
	}

	// 5. Test redis get on JSON key (auto-formatting)
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"get", "test:key:json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get JSON key: %v", err)
	}
	if !strings.Contains(buf.String(), `"count": 42`) {
		t.Fatalf("expected formatted JSON, got %q", buf.String())
	}

	// 6. Test redis show
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"show", "--pattern", "test:key:*", "--limit", "10"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run show: %v", err)
	}
	showOut := buf.String()
	if !strings.Contains(showOut, "test:key:string") || !strings.Contains(showOut, "test:key:json") {
		t.Fatalf("expected show to list seeded keys, got: %s", showOut)
	}

	// 7. Test redis show --json
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"show", "--pattern", "test:key:*", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run show --json: %v", err)
	}
	if !strings.Contains(buf.String(), `"key": "test:key:string"`) {
		t.Fatalf("expected JSON keys in show output, got: %s", buf.String())
	}

	// 8. Test redis inspect
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"inspect", "test:key:json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run inspect: %v", err)
	}
	inspectOut := buf.String()
	if !strings.Contains(inspectOut, "test:key:json") || !strings.Contains(inspectOut, "STRING") {
		t.Fatalf("expected inspect card with key and type, got: %s", inspectOut)
	}

	// 9. Test redis get on non-existent key returns ExitKeyNotFound
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"get", "test:key:does_not_exist"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error on non-existent key, got nil")
	}

	// 10. Test redis set with --json flag
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"set", "test:key:json_flag", "test_val", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to execute set with --json: %v", err)
	}
	if !strings.Contains(buf.String(), `"status": "OK"`) {
		t.Fatalf("expected JSON response from set, got %q", buf.String())
	}
}
