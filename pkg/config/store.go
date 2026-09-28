package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPath returns $XDG_CONFIG_HOME/redis/contexts.yaml or ~/.config/redis/contexts.yaml
func DefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.Getenv("HOME"), ".config", "redis", "contexts.yaml")
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "redis", "contexts.yaml")
}

// Load loads the configuration from the specified file path.
// It returns an empty config (not nil) and os.ErrNotExist if the file doesn't exist.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Contexts: make([]Context, 0)}, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// 0600 permission check
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}
	if info.Mode().Perm() != 0600 {
		return nil, fmt.Errorf("config file %s has insecure permissions %04o, must be 0600", path, info.Mode().Perm())
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Environment variable interpolation
	expandedContent := os.ExpandEnv(string(content))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expandedContent), cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = make([]Context, 0)
	}

	return cfg, nil
}

// Save saves the configuration to the specified file path atomically.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, "redis-cli-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up tmp file in case of early return/error
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	// Ensure 0600 permissions
	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Lock the temporary file
	if err := syscall.Flock(int(tmpFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock temporary file: %w", err)
	}
	defer syscall.Flock(int(tmpFile.Fd()), syscall.LOCK_UN)

	// Write data
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the file before renaming
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temporary file to target path: %w", err)
	}

	return nil
}
