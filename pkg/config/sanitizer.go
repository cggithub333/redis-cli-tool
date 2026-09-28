package config

import (
	"fmt"
	"regexp"
	"strings"
)

var nonAlphaNumRegex = regexp.MustCompile(`[^A-Za-z0-9_]`)

// SanitizeContext creates a copy of Context with secret fields masked as ${ENV_VAR} placeholders.
func SanitizeContext(c *Context) *Context {
	if c == nil {
		return nil
	}

	cpy := *c
	if cpy.Password != "" {
		// If already an env var placeholder like ${...}, preserve it
		if strings.HasPrefix(cpy.Password, "${") && strings.HasSuffix(cpy.Password, "}") {
			return &cpy
		}

		cleanName := strings.ToUpper(nonAlphaNumRegex.ReplaceAllString(cpy.Name, "_"))
		if cleanName == "" {
			cleanName = "REDIS"
		}
		cpy.Password = fmt.Sprintf("${%s_PASSWORD}", cleanName)
	}

	return &cpy
}

// SanitizeConfig returns a cloned Config with all context secrets masked.
func SanitizeConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	sanitized := &Config{
		CurrentContext: cfg.CurrentContext,
		Contexts:       make([]Context, len(cfg.Contexts)),
	}

	for i, ctx := range cfg.Contexts {
		sCtx := SanitizeContext(&ctx)
		sanitized.Contexts[i] = *sCtx
	}

	return sanitized
}
