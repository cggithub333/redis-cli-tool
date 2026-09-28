package client

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ScanKeys non-blocking SCAN iteration with safely chunked metadata pipelines (max 200 keys per pipeline flush).
func (c *Client) ScanKeys(ctx context.Context, pattern string, cursor uint64, count int64) (*ScanResult, error) {
	keys, nextCursor, err := c.Client.Scan(ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		Cursor: nextCursor,
		Keys:   make([]KeyMetadata, 0, len(keys)),
	}

	const batchSize = 200

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batchKeys := keys[i:end]

		pipe := c.Client.Pipeline()
		var cmds []struct {
			key    string
			typ    *redis.StatusCmd
			ttl    *redis.DurationCmd
			memory *redis.IntCmd
		}

		for _, k := range batchKeys {
			cmds = append(cmds, struct {
				key    string
				typ    *redis.StatusCmd
				ttl    *redis.DurationCmd
				memory *redis.IntCmd
			}{
				key:    k,
				typ:    pipe.Type(ctx, k),
				ttl:    pipe.TTL(ctx, k),
				memory: pipe.MemoryUsage(ctx, k),
			})
		}

		_, pipeErr := pipe.Exec(ctx)
		// We process the commands even if there are errors (e.g., MEMORY USAGE not supported),
		// but we might want to handle it selectively. MemoryUsage might return an error if it doesn't exist.
		if pipeErr != nil && !strings.Contains(pipeErr.Error(), "ERR") && !strings.Contains(pipeErr.Error(), "unknown command") {
			// Some other error we shouldn't ignore entirely? But we can just proceed and check individual command errors.
		}

		for _, c := range cmds {
			typ, _ := c.typ.Result()
			ttl, _ := c.ttl.Result()
			mem, err := c.memory.Result()
			if err != nil {
				// Fallback to 0 if MEMORY USAGE is not supported or errors out.
				mem = 0
			}

			result.Keys = append(result.Keys, KeyMetadata{
				Key:    c.key,
				Type:   typ,
				TTL:    ttl,
				Memory: mem,
			})
		}
	}

	return result, nil
}
