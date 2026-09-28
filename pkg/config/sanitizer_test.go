package config

import (
	"testing"
)

func TestSanitizeContext(t *testing.T) {
	tests := []struct {
		name     string
		input    Context
		expected string
	}{
		{
			name: "plain password masked to env placeholder",
			input: Context{
				Name:     "local-common",
				Host:     "127.0.0.1",
				Port:     6379,
				Password: "supersecretpassword",
			},
			expected: "${LOCAL_COMMON_PASSWORD}",
		},
		{
			name: "already env placeholder unchanged",
			input: Context{
				Name:     "prod-cluster",
				Host:     "redis.prod",
				Port:     6379,
				Password: "${MY_SECRET_PWD}",
			},
			expected: "${MY_SECRET_PWD}",
		},
		{
			name: "empty password unchanged",
			input: Context{
				Name:     "dev",
				Host:     "localhost",
				Port:     6379,
				Password: "",
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := SanitizeContext(&tc.input)
			if sanitized.Password != tc.expected {
				t.Errorf("expected password %q, got %q", tc.expected, sanitized.Password)
			}
			// Verify original not mutated
			if tc.input.Password != "" && tc.input.Password != tc.expected && tc.input.Password == sanitized.Password {
				t.Errorf("original context password was mutated")
			}
		})
	}
}

func TestSanitizeConfig(t *testing.T) {
	cfg := &Config{
		CurrentContext: "c1",
		Contexts: []Context{
			{Name: "c1", Host: "127.0.0.1", Port: 6379, Password: "secret1"},
			{Name: "c2", Host: "127.0.0.1", Port: 6380, Password: "secret2"},
		},
	}

	sanitized := SanitizeConfig(cfg)
	if sanitized.Contexts[0].Password != "${C1_PASSWORD}" {
		t.Errorf("expected ${C1_PASSWORD}, got %q", sanitized.Contexts[0].Password)
	}
	if sanitized.Contexts[1].Password != "${C2_PASSWORD}" {
		t.Errorf("expected ${C2_PASSWORD}, got %q", sanitized.Contexts[1].Password)
	}
}
