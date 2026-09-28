package cmd

import (
	"fmt"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
	"redis-cli-tool/pkg/format"
)

// getActiveClient resolves the target context and constructs a connected *client.Client
func getActiveClient() (*client.Client, config.Context, error) {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, config.Context{}, fmt.Errorf("failed to load context configuration: %w", err)
	}

	targetCtx, err := config.ResolveContext(cfg, contextFlag)
	if err != nil {
		return nil, config.Context{}, err
	}

	cl, err := client.NewClient(&targetCtx)
	if err != nil {
		return nil, targetCtx, fmt.Errorf("failed to initialize Redis client [%s:%d]: %w", targetCtx.Host, targetCtx.Port, err)
	}

	return cl, targetCtx, nil
}

// formatBytes delegates to the canonical format.FormatBytes implementation
func formatBytes(bytes int64) string {
	return format.FormatBytes(bytes)
}
