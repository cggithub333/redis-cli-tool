package config

// Context represents a Redis connection context
type Context struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	DB       int    `yaml:"db,omitempty"`
	TLS      bool   `yaml:"tls,omitempty"`
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

