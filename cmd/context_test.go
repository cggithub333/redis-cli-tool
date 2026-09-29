package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextCRUD(t *testing.T) {
	// Set isolated test XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// 1. Initially empty
	ResetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"context", "ls"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error on empty ls: %v", err)
	}

	// 2. Create context local-test-1
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "create", "local-test-1", "--host", "127.0.0.1", "--port", "6379", "--password", "secret123"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context: %v", err)
	}
	if !strings.Contains(buf.String(), `Context "local-test-1" created.`) {
		t.Fatalf("expected creation message, got: %s", buf.String())
	}

	// 3. Create context local-test-2
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "create", "local-test-2", "--host", "127.0.0.1", "--port", "6380"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create second context: %v", err)
	}

	// 4. Verify current context defaults to the first created
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "current"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get current context: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "local-test-1" {
		t.Fatalf("expected current context local-test-1, got %q", buf.String())
	}

	// 5. Switch to local-test-2
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "use", "local-test-2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to use context: %v", err)
	}

	// 6. Verify current context updated
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "current"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get current context after switch: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "local-test-2" {
		t.Fatalf("expected current context local-test-2, got %q", buf.String())
	}

	// 7. Context ls JSON output
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "ls", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run context ls --json: %v", err)
	}
	jsonOut := buf.String()
	if !strings.Contains(jsonOut, `"name": "local-test-1"`) || !strings.Contains(jsonOut, `"name": "local-test-2"`) {
		t.Fatalf("expected both contexts in JSON output: %s", jsonOut)
	}

	// 8. Context export (sanitized)
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "export"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to export contexts: %v", err)
	}
	exportOut := buf.String()
	if strings.Contains(exportOut, "secret123") {
		t.Fatalf("expected exported password to be masked, but found plaintext in: %s", exportOut)
	}
	if !strings.Contains(exportOut, "${LOCAL_TEST_1_PASSWORD}") {
		t.Fatalf("expected ${LOCAL_TEST_1_PASSWORD} placeholder in export, got: %s", exportOut)
	}

	// 9. Context export with secrets
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "export", "--include-secrets"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to export with secrets: %v", err)
	}
	if !strings.Contains(buf.String(), "secret123") {
		t.Fatalf("expected plaintext secret when --include-secrets is passed, got: %s", buf.String())
	}

	// 10. Delete context local-test-1
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "delete", "local-test-1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to delete context: %v", err)
	}

	// Verify local-test-1 is gone
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "ls", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run context ls: %v", err)
	}
	if strings.Contains(buf.String(), `"name": "local-test-1"`) {
		t.Fatalf("expected local-test-1 to be deleted, but still found in: %s", buf.String())
	}

	// 11. Import contexts from exported file
	importFile := filepath.Join(tmpDir, "import.yaml")
	importContent := `current-context: imported-ctx
contexts:
  - name: imported-ctx
    host: 127.0.0.1
    port: 6379
`
	if err := os.WriteFile(importFile, []byte(importContent), 0600); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "import", importFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to import contexts: %v", err)
	}

	// Verify imported context is active
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "current"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get current context after import: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "imported-ctx" {
		t.Fatalf("expected imported-ctx to be current, got %q", buf.String())
	}
}

func TestContextCreateURI(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// 1. Cloud TLS URI
	ResetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"context", "create", "cloud-upstash", "--uri", "rediss://default:upstashpass@us1-test.upstash.io:6379/1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context with cloud TLS URI: %v", err)
	}

	// Verify created context
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "export", "--include-secrets"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to export contexts: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "name: cloud-upstash") ||
		!strings.Contains(out, "host: us1-test.upstash.io") ||
		!strings.Contains(out, "port: 6379") ||
		!strings.Contains(out, "username: default") ||
		!strings.Contains(out, "password: upstashpass") ||
		!strings.Contains(out, "tls: true") ||
		!strings.Contains(out, "db: 1") {
		t.Fatalf("cloud context attributes not parsed properly: %s", out)
	}

	// 2. Override flag with URI
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "create", "cloud-override", "--url", "redis://:localpass@127.0.0.1:6379/0", "--port", "6385"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context with override: %v", err)
	}

	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "export", "--include-secrets"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to export contexts: %v", err)
	}
	if !strings.Contains(buf.String(), "port: 6385") {
		t.Fatalf("expected port flag to override URI port, got: %s", buf.String())
	}

	// 3. Auto-derived name and HTTP URI normalization
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "create", "--uri", "http://localhost:6379"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context with http URI and omitted name: %v", err)
	}

	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "export"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to export contexts: %v", err)
	}
	if !strings.Contains(buf.String(), "name: local-6379") {
		t.Fatalf("expected auto-derived name 'local-6379', got: %s", buf.String())
	}

	// 4. Invalid URI error
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "create", "bad-uri", "--uri", "invalid://bad-scheme"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error on invalid URI scheme, got nil")
	}
}

func TestContextRenameAndSupplier(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// 1. Create context with auto-detected Layerbase supplier
	ResetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"context", "create", "layerbase", "--host", "127.0.0.1", "--port", "6379", "--supplier", "Layerbase"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	// 2. Check context ls table has SUPPLIER column
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "ls"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to list contexts: %v", err)
	}
	lsOut := buf.String()
	if !strings.Contains(lsOut, "SUPPLIER") || !strings.Contains(lsOut, "Layerbase") {
		t.Fatalf("expected SUPPLIER column and Layerbase row, got: %s", lsOut)
	}

	// 3. Rename context from layerbase to capstone-redis-001
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "rename", "layerbase", "capstone-redis-001"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to rename context: %v", err)
	}
	if !strings.Contains(buf.String(), `Renamed context "layerbase" to "capstone-redis-001".`) {
		t.Fatalf("expected rename message, got: %s", buf.String())
	}

	// 4. Verify current context updated
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "current"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to get current context: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "capstone-redis-001" {
		t.Fatalf("expected current context capstone-redis-001, got %q", buf.String())
	}

	// 5. Verify old context no longer exists
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "use", "layerbase"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error switching to deleted old context name, got nil")
	}

	// 6. Rename with supplier update
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "rename", "capstone-redis-001", "capstone-official", "--supplier", "Redis Official"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to rename with supplier update: %v", err)
	}

	// Check ls output shows Redis Official
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "ls"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to list contexts: %v", err)
	}
	if !strings.Contains(buf.String(), "Redis Official") {
		t.Fatalf("expected updated supplier Redis Official, got: %s", buf.String())
	}

	// 7. Rename non-existent error
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"context", "rename", "does-not-exist", "foo"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error renaming non-existent context, got nil")
	}
}


