package client

import (
	"context"
	"testing"

	"redis-cli-tool/pkg/config"
)

func TestClient_Ping(t *testing.T) {
	cfg := &config.Context{
		Host: "127.0.0.1",
		Port: 6379,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	latency, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if latency <= 0 {
		t.Errorf("Expected positive latency, got %v", latency)
	}
}
