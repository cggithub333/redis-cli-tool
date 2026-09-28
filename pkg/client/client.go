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

// Ping measures latency to the Redis server.
func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
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
