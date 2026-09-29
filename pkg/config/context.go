package config

import "strings"

// Context represents a Redis connection context
type Context struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	DB       int    `yaml:"db,omitempty"`
	TLS      bool   `yaml:"tls,omitempty"`
	Supplier string `yaml:"supplier,omitempty"`
}

// Config represents the complete CLI configuration
type Config struct {
	CurrentContext string    `yaml:"current-context"`
	Contexts       []Context `yaml:"contexts"`
}

// GetContext retrieves a context by name
func (cfg *Config) GetContext(name string) (*Context, bool) {
	if cfg == nil {
		return nil, false
	}
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == name {
			return &cfg.Contexts[i], true
		}
	}
	return nil, false
}

// SetContext adds or updates a context by name
func (cfg *Config) SetContext(ctx Context) {
	if cfg == nil {
		return
	}
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == ctx.Name {
			cfg.Contexts[i] = ctx
			return
		}
	}
	cfg.Contexts = append(cfg.Contexts, ctx)
}

// DeleteContext removes a context by name, clearing CurrentContext if it matches
func (cfg *Config) DeleteContext(name string) bool {
	if cfg == nil {
		return false
	}
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == name {
			cfg.Contexts = append(cfg.Contexts[:i], cfg.Contexts[i+1:]...)
			if cfg.CurrentContext == name {
				cfg.CurrentContext = ""
			}
			return true
		}
	}
	return false
}

// RenameContext renames an existing context, updating CurrentContext if it matches
func (cfg *Config) RenameContext(oldName, newName string) bool {
	if cfg == nil {
		return false
	}
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == oldName {
			cfg.Contexts[i].Name = newName
			if cfg.CurrentContext == oldName {
				cfg.CurrentContext = newName
			}
			return true
		}
	}
	return false
}

// DetectSupplier infers or normalizes the Redis supplier / provider.
func DetectSupplier(host, explicit string) string {
	if explicit != "" {
		return explicit
	}
	lowerHost := strings.ToLower(host)
	switch {
	case strings.Contains(lowerHost, "layerbase"):
		return "Layerbase"
	case strings.Contains(lowerHost, "upstash"):
		return "Upstash"
	case strings.Contains(lowerHost, "redislabs.com") || strings.Contains(lowerHost, "redis.com") || strings.Contains(lowerHost, "redis.io"):
		return "Redis Official"
	case strings.Contains(lowerHost, "elasticache") || strings.Contains(lowerHost, "amazonaws.com"):
		return "AWS ElastiCache"
	case strings.Contains(lowerHost, "aiven"):
		return "Aiven"
	case lowerHost == "127.0.0.1" || lowerHost == "localhost" || lowerHost == "::1":
		return "Local container"
	default:
		return "Custom"
	}
}

