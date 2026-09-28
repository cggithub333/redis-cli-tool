package client

import "time"

// KeyMetadata contains metadata for a specific Redis key.
type KeyMetadata struct {
	Key    string
	Type   string
	TTL    time.Duration
	Memory int64
}

// ScanResult holds a batch of scanned keys and the next cursor.
type ScanResult struct {
	Keys   []KeyMetadata
	Cursor uint64
}
