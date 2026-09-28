package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"redis-cli-tool/pkg/config"
)

func TestScanner_ScanKeys(t *testing.T) {
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

	// Setup some keys
	for i := 0; i < 250; i++ {
		key := fmt.Sprintf("test_scan_key:%d", i)
		err := client.Client.Set(ctx, key, "value", time.Minute).Err()
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}
	}
	defer func() {
		// Cleanup
		for i := 0; i < 250; i++ {
			client.Client.Del(ctx, fmt.Sprintf("test_scan_key:%d", i))
		}
	}()

	// Perform scan
	res, err := client.ScanKeys(ctx, "test_scan_key:*", 0, 500)
	if err != nil {
		t.Fatalf("ScanKeys failed: %v", err)
	}

	if len(res.Keys) == 0 {
		t.Errorf("Expected to find some keys, found 0")
	}

	// Verify metadata
	for _, km := range res.Keys {
		if km.Type != "string" {
			t.Errorf("Expected type string for key %s, got %s", km.Key, km.Type)
		}
		if km.TTL <= 0 {
			t.Errorf("Expected positive TTL for key %s, got %v", km.Key, km.TTL)
		}
		// Memory might be 0 if the version doesn't support it or if it errors, but typically it should be > 0.
		// We'll just verify it doesn't panic.
	}
}
