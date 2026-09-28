package config

import (
	"fmt"
	"os"
)

// ResolveContext resolves the context to use based on the 3-tier precedence:
// 1. --context flag (if provided)
// 2. REDIS_CONTEXT environment variable (if set)
// 3. current-context from config file
func ResolveContext(cfg *Config, flagContext string) (Context, error) {
	ctxName := flagContext

	if ctxName == "" {
		ctxName = os.Getenv("REDIS_CONTEXT")
	}

	if ctxName == "" && cfg != nil {
		ctxName = cfg.CurrentContext
	}

	if ctxName == "" {
		return Context{}, fmt.Errorf("no context specified (use --context, REDIS_CONTEXT, or set current-context in config)")
	}

	if cfg != nil {
		for _, ctx := range cfg.Contexts {
			if ctx.Name == ctxName {
				return ctx, nil
			}
		}
	}

	return Context{}, fmt.Errorf("context '%s' not found", ctxName)
}
