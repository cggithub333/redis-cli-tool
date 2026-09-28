package config

import (
	"os"
	"testing"
)

func TestResolveContext(t *testing.T) {
	cfg := &Config{
		CurrentContext: "default",
		Contexts: []Context{
			{Name: "default", Host: "localhost"},
			{Name: "env", Host: "env.local"},
			{Name: "flag", Host: "flag.local"},
		},
	}

	t.Run("FlagPrecedence", func(t *testing.T) {
		os.Setenv("REDIS_CONTEXT", "env")
		defer os.Unsetenv("REDIS_CONTEXT")

		ctx, err := ResolveContext(cfg, "flag")
		if err != nil {
			t.Fatalf("ResolveContext failed: %v", err)
		}
		if ctx.Name != "flag" {
			t.Errorf("Expected 'flag', got '%s'", ctx.Name)
		}
	})

	t.Run("EnvPrecedence", func(t *testing.T) {
		os.Setenv("REDIS_CONTEXT", "env")
		defer os.Unsetenv("REDIS_CONTEXT")

		ctx, err := ResolveContext(cfg, "")
		if err != nil {
			t.Fatalf("ResolveContext failed: %v", err)
		}
		if ctx.Name != "env" {
			t.Errorf("Expected 'env', got '%s'", ctx.Name)
		}
	})

	t.Run("ConfigPrecedence", func(t *testing.T) {
		ctx, err := ResolveContext(cfg, "")
		if err != nil {
			t.Fatalf("ResolveContext failed: %v", err)
		}
		if ctx.Name != "default" {
			t.Errorf("Expected 'default', got '%s'", ctx.Name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := ResolveContext(cfg, "missing")
		if err == nil {
			t.Fatal("Expected error for missing context, got nil")
		}
	})

	t.Run("NoContext", func(t *testing.T) {
		emptyCfg := &Config{}
		_, err := ResolveContext(emptyCfg, "")
		if err == nil {
			t.Fatal("Expected error for no context, got nil")
		}
	})
}
