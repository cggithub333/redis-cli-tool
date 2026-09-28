package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		CurrentContext: "prod",
		Contexts: []Context{
			{Name: "prod", Host: "redis.prod.internal", Port: 6379},
		},
	}

	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %04o", info.Mode().Perm())
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.CurrentContext != "prod" {
		t.Errorf("Expected current-context 'prod', got '%s'", loaded.CurrentContext)
	}
	if len(loaded.Contexts) != 1 || loaded.Contexts[0].Name != "prod" {
		t.Errorf("Expected 1 context named 'prod', got %v", loaded.Contexts)
	}
}

func TestStore_InsecurePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	os.WriteFile(configPath, []byte("current-context: dev\n"), 0644)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Expected error due to insecure permissions, got nil")
	}
}

func TestStore_EnvInterpolation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlData := `
current-context: dev
contexts:
  - name: dev
    host: ${REDIS_HOST}
    port: 6379
    password: ${REDIS_PASS}
`
	os.WriteFile(configPath, []byte(yamlData), 0600)

	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PASS", "secret")
	defer os.Unsetenv("REDIS_HOST")
	defer os.Unsetenv("REDIS_PASS")

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Contexts[0].Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", loaded.Contexts[0].Host)
	}
	if loaded.Contexts[0].Password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", loaded.Contexts[0].Password)
	}
}
