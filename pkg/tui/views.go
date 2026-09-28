package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	paneHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				Background(lipgloss.Color("237"))

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

func (m Model) View() string {
	if m.Width > 0 && (m.Width < 60 || m.Height < 15) {
		return fmt.Sprintf("Terminal window is too small (%dx%d).\nPlease resize to at least 60x15.", m.Width, m.Height)
	}

	leftWidth := 35
	if m.Width > 90 {
		leftWidth = m.Width * 35 / 100
	}
	rightWidth := m.Width - leftWidth - 5
	if rightWidth < 30 {
		rightWidth = 30
	}

	// 1. Left Pane: Key List
	var leftLines []string
	leftLines = append(leftLines, paneHeaderStyle.Render(fmt.Sprintf("KEYS (%d)", len(m.FilteredKeys))))

	if m.Searching {
		leftLines = append(leftLines, "/ "+m.SearchInput.View())
	}

	maxDisplay := m.Height - 6
	if maxDisplay < 5 {
		maxDisplay = 5
	}

	start := 0
	if m.Cursor >= maxDisplay {
		start = m.Cursor - maxDisplay + 1
	}
	end := start + maxDisplay
	if end > len(m.FilteredKeys) {
		end = len(m.FilteredKeys)
	}

	for i := start; i < end; i++ {
		k := m.FilteredKeys[i]
		keyStr := k.Key
		if len(keyStr) > leftWidth-10 {
			keyStr = keyStr[:leftWidth-13] + "..."
		}

		line := fmt.Sprintf("%-6s %s", k.Type, keyStr)
		if i == m.Cursor {
			leftLines = append(leftLines, selectedItemStyle.Render("> "+line))
		} else {
			leftLines = append(leftLines, normalItemStyle.Render("  "+line))
		}
	}

	if len(m.FilteredKeys) == 0 {
		leftLines = append(leftLines, "  (No keys matching)")
	}

	leftPane := lipgloss.NewStyle().
		Width(leftWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(lipgloss.Color("240")).
		Render(strings.Join(leftLines, "\n"))

	// 2. Right Pane: Value & Metadata Preview
	var rightLines []string
	if len(m.FilteredKeys) > 0 && m.Cursor < len(m.FilteredKeys) {
		curr := m.FilteredKeys[m.Cursor]
		metaCard := fmt.Sprintf("KEY: %s\nTYPE: %s  TTL: %s  MEM: %s",
			curr.Key, m.SelectedType, m.SelectedTTL, m.SelectedMem)
		rightLines = append(rightLines, cardBorder.Render(metaCard))
		rightLines = append(rightLines, "\nVALUE PREVIEW:")
		rightLines = append(rightLines, m.SelectedValue)
	} else {
		rightLines = append(rightLines, "Select a key to view value")
	}

	rightPane := lipgloss.NewStyle().
		Width(rightWidth).
		Padding(0, 1).
		Render(strings.Join(rightLines, "\n"))

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// 3. Status Bar
	status := fmt.Sprintf("Context: %s | Keys: %d | [/] Search  [j/k] Navigate  [q] Quit",
		m.ContextName, len(m.Keys))
	if m.StatusMsg != "" {
		status += " | " + m.StatusMsg
	}
	footer := statusBarStyle.Width(m.Width).Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("REDIS EXPLORER"), mainContent, footer)
}
