package config

import (
	"testing"
)

func TestContext_Rename(t *testing.T) {
	cfg := &Config{
		CurrentContext: "old-ctx",
		Contexts: []Context{
			{Name: "old-ctx", Host: "127.0.0.1", Port: 6379, Supplier: "Local container"},
			{Name: "other-ctx", Host: "127.0.0.1", Port: 6380},
		},
	}

	// Rename existing context
	ok := cfg.RenameContext("old-ctx", "new-ctx")
	if !ok {
		t.Fatalf("expected RenameContext to return true, got false")
	}

	if cfg.CurrentContext != "new-ctx" {
		t.Errorf("expected CurrentContext to update to 'new-ctx', got '%s'", cfg.CurrentContext)
	}

	ctx, exists := cfg.GetContext("new-ctx")
	if !exists || ctx.Name != "new-ctx" {
		t.Errorf("expected context 'new-ctx' to exist, got %v", ctx)
	}

	if _, exists := cfg.GetContext("old-ctx"); exists {
		t.Errorf("expected 'old-ctx' to no longer exist")
	}

	// Rename non-existent context
	if ok := cfg.RenameContext("non-existent", "fail"); ok {
		t.Errorf("expected RenameContext on non-existent to return false")
	}
}

func TestContext_DetectSupplier(t *testing.T) {
	tests := []struct {
		host     string
		explicit string
		expected string
	}{
		{"127.0.0.1", "", "Local container"},
		{"localhost", "", "Local container"},
		{"::1", "", "Local container"},
		{"capstone-redis-001-west-bow.sage.cloud.layerbase.dev", "", "Layerbase"},
		{"us1-proud-cat-12345.upstash.io", "", "Upstash"},
		{"redis-12345.c1.us-central1-1.gce.redislabs.com", "", "Redis Official"},
		{"my-cluster.abcdef.0001.use1.cache.amazonaws.com", "", "AWS ElastiCache"},
		{"redis-service.aivencloud.com", "", "Aiven"},
		{"custom.host.internal", "", "Custom"},
		{"custom.host.internal", "My Company Private", "My Company Private"},
	}

	for _, tt := range tests {
		got := DetectSupplier(tt.host, tt.explicit)
		if got != tt.expected {
			t.Errorf("DetectSupplier(%q, %q) = %q, expected %q", tt.host, tt.explicit, got, tt.expected)
		}
	}
}
