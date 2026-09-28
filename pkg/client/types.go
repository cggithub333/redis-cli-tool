package client

import "time"

// KeyMetadata contains metadata for a specific Redis key.
type KeyMetadata struct {
	Key    string        `json:"key"`
	Type   string        `json:"type"`
	TTL    time.Duration `json:"ttl"`
	TTLStr string        `json:"ttl_str,omitempty"`
	Memory int64         `json:"memory_bytes"`
}

// ScanResult holds a batch of scanned keys and the next cursor.
type ScanResult struct {
	Keys   []KeyMetadata `json:"keys"`
	Cursor uint64        `json:"cursor"`
}
