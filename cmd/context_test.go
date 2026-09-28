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
