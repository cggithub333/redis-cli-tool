package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/format"
)

type Model struct {
	Client        *client.Client
	ContextName   string
	Keys          []client.KeyMetadata
	FilteredKeys  []client.KeyMetadata
	Cursor        int
	Width         int
	Height        int
	SearchInput   textinput.Model
	Searching     bool
	SelectedValue string
	SelectedType  string
	SelectedTTL   string
	SelectedMem   string
	StatusMsg     string
	KeyMap        KeyMap
}

func NewModel(cl *client.Client, ctxName string, initialKeys []client.KeyMetadata) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 100

	m := Model{
		Client:       cl,
		ContextName:  ctxName,
		Keys:         initialKeys,
		FilteredKeys: initialKeys,
		Cursor:       0,
		SearchInput:  ti,
		KeyMap:       DefaultKeyMap,
	}
	m.UpdatePreview()
	return m
}

func (m *Model) FilterKeys(query string) {
	if query == "" {
		m.FilteredKeys = m.Keys
		m.Cursor = 0
		m.UpdatePreview()
		return
	}

	q := strings.ToLower(query)
	var filtered []client.KeyMetadata
	for _, k := range m.Keys {
		if strings.Contains(strings.ToLower(k.Key), q) {
			filtered = append(filtered, k)
		}
	}
	m.FilteredKeys = filtered
	m.Cursor = 0
	m.UpdatePreview()
}

func (m *Model) UpdatePreview() {
	if len(m.FilteredKeys) == 0 || m.Cursor >= len(m.FilteredKeys) {
		m.SelectedValue = "No keys available"
		m.SelectedType = "-"
		m.SelectedTTL = "-"
		m.SelectedMem = "-"
		return
	}

	curr := m.FilteredKeys[m.Cursor]
	m.SelectedType = strings.ToUpper(curr.Type)
	m.SelectedTTL = curr.TTLStr
	m.SelectedMem = format.FormatBytes(curr.Memory)

	if m.Client == nil {
		m.SelectedValue = "(Key details available in connected session)"
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	decoded, err := format.DecodeRedisKey(ctx, m.Client, curr.Key, curr.Type, false)
	if err != nil {
		m.SelectedValue = fmt.Sprintf("Error reading key: %v", err)
		return
	}

	m.SelectedValue = decoded.Formatted
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.Searching {
			switch msg.Type {
			case tea.KeyEnter:
				m.Searching = false
				m.SearchInput.Blur()
				m.FilterKeys(m.SearchInput.Value())
				return m, nil
			case tea.KeyEsc:
				m.Searching = false
				m.SearchInput.Blur()
				m.SearchInput.SetValue("")
				m.FilterKeys("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.SearchInput, cmd = m.SearchInput.Update(msg)
				m.FilterKeys(m.SearchInput.Value())
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.KeyMap.Up):
			if m.Cursor > 0 {
				m.Cursor--
				m.UpdatePreview()
			}
			return m, nil

		case key.Matches(msg, m.KeyMap.Down):
			if m.Cursor < len(m.FilteredKeys)-1 {
				m.Cursor++
				m.UpdatePreview()
			}
			return m, nil

		case key.Matches(msg, m.KeyMap.Search):
			m.Searching = true
			m.SearchInput.Focus()
			return m, textinput.Blink

		case key.Matches(msg, m.KeyMap.Esc):
			if m.SearchInput.Value() != "" {
				m.SearchInput.SetValue("")
				m.FilterKeys("")
			}
			return m, nil
		}
	}

	return m, nil
}
