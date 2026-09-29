package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"redis-cli-tool/pkg/config"
)

// Client wraps the go-redis Client and the configuration context.
type Client struct {
	*redis.Client
	Config *config.Context
}

// NewClient constructs a new Redis client based on the provided config context.
func NewClient(cfg *config.Context) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration context cannot be nil")
	}
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.TLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)

	return &Client{
		Client: rdb,
		Config: cfg,
	}, nil
}

// Ping measures true round-trip latency to the Redis server over an established connection.
// It performs a warmup ping first (which handles cold TCP dial, TLS handshake, and AUTH),
// then precisely measures the subsequent Redis command response round-trip time.
func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
	// First ping triggers lazy dial, TLS handshake, and Redis AUTH
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return 0, err
	}

	// Second ping measures true Redis command round-trip latency over the warm socket
	start := time.Now()
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// Close closes the underlying Redis client connection.
func (c *Client) Close() error {
	return c.Client.Close()
}
