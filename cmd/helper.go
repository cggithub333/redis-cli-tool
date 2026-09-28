package cmd

import (
	"fmt"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
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

// formatBytes formats raw byte count into human readable B, KB, MB, GB string
func formatBytes(bytes int64) string {
	if bytes < 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
