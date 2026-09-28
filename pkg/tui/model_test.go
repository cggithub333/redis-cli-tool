package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"redis-cli-tool/pkg/client"
)

func TestTUIModel(t *testing.T) {
	keys := []client.KeyMetadata{
		{Key: "user:1", Type: "string", TTL: time.Hour, TTLStr: "1h", Memory: 64},
		{Key: "user:2", Type: "string", TTL: time.Hour, TTLStr: "1h", Memory: 64},
		{Key: "config:app", Type: "hash", TTL: -1, TTLStr: "none", Memory: 128},
	}

	m := NewModel(nil, "test-context", keys)
	if len(m.FilteredKeys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(m.FilteredKeys))
	}
	if m.Cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.Cursor)
	}

	// Navigate down
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mModel.(Model)
	if m.Cursor != 1 {
		t.Fatalf("expected cursor 1 after 'j', got %d", m.Cursor)
	}

	// Filter keys
	m.FilterKeys("config")
	if len(m.FilteredKeys) != 1 {
		t.Fatalf("expected 1 filtered key matching 'config', got %d", len(m.FilteredKeys))
	}
	if m.FilteredKeys[0].Key != "config:app" {
		t.Fatalf("expected 'config:app', got %s", m.FilteredKeys[0].Key)
	}
}

func TestTUIViews(t *testing.T) {
	keys := []client.KeyMetadata{
		{Key: "test:key", Type: "string", TTLStr: "none", Memory: 100},
	}
	m := NewModel(nil, "test-context", keys)

	// Small terminal check
	m.Width = 40
	m.Height = 10
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "too small") {
		t.Fatalf("expected small terminal warning, got: %s", viewSmall)
	}

	// Normal terminal check
	m.Width = 100
	m.Height = 30
	viewNormal := m.View()
	if !strings.Contains(viewNormal, "REDIS EXPLORER") {
		t.Fatalf("expected title in normal view, got: %s", viewNormal)
	}
	if !strings.Contains(viewNormal, "test:key") {
		t.Fatalf("expected key in normal view, got: %s", viewNormal)
	}
}
