package format

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5722")).
			Padding(0, 1)

	RowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	BorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	HealthyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	UnreachableStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	ActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFA726"))

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))
)

// StatusBadge returns a styled status string with an indicator icon.
func StatusBadge(status string) string {
	upper := strings.ToUpper(strings.TrimSpace(status))
	switch upper {
	case "HEALTHY", "ONLINE", "OK":
		return HealthyStyle.Render("● " + upper)
	case "UNREACHABLE", "OFFLINE", "ERROR", "DOWN":
		return UnreachableStyle.Render("○ " + upper)
	case "ACTIVE":
		return ActiveStyle.Render("★ " + upper)
	default:
		return WarningStyle.Render("? " + upper)
	}
}

// RenderTable renders a structured ASCII/Unicode border table using Lipgloss.
func RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 && len(rows) == 0 {
		return ""
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(BorderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return HeaderStyle
			}
			return RowStyle
		})

	return t.Render()
}

// FormatBytes formats byte counts into human readable strings (B, KB, MB, GB)
func FormatBytes(bytes int64) string {
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
