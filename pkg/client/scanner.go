package client

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ScanKeys non-blocking SCAN iteration with safely chunked metadata pipelines (max 200 keys per pipeline flush).
func (c *Client) ScanKeys(ctx context.Context, pattern string, cursor uint64, count int64) (*ScanResult, error) {
	targetCount := count
	if targetCount <= 0 {
		targetCount = 50
	}

	var allKeys []string
	currCursor := cursor

	for {
		batchKeys, nextCursor, err := c.Client.Scan(ctx, currCursor, pattern, targetCount).Result()
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, batchKeys...)
		currCursor = nextCursor

		if currCursor == 0 || len(allKeys) >= int(targetCount) {
			break
		}
	}

	if len(allKeys) > int(targetCount) {
		allKeys = allKeys[:targetCount]
	}

	result := &ScanResult{
		Cursor: currCursor,
		Keys:   make([]KeyMetadata, 0, len(allKeys)),
	}

	const batchSize = 200

	for i := 0; i < len(allKeys); i += batchSize {
		end := i + batchSize
		if end > len(allKeys) {
			end = len(allKeys)
		}

		batchKeys := allKeys[i:end]

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
		if pipeErr != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			errStr := pipeErr.Error()
			if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "broken pipe") || strings.Contains(errStr, "i/o timeout") {
				return nil, pipeErr
			}
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
