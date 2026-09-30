package format

import (
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// DetectTerminalWidth detects the available terminal or preview pane width.
func DetectTerminalWidth() int {
	// 1. Check FZF preview columns (set by fzf in preview subprocess)
	if fzfCols := os.Getenv("FZF_PREVIEW_COLUMNS"); fzfCols != "" {
		if w, err := strconv.Atoi(fzfCols); err == nil && w > 20 {
			return w
		}
	}

	// 2. Check COLUMNS environment variable
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if w, err := strconv.Atoi(cols); err == nil && w > 20 {
			return w
		}
	}

	// 3. Query terminal size from Stdout if TTY
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 20 {
			return w
		}
	}

	return 80
}

// GetLanguageBadgeColor returns a distinctive accent color for the language badge.
func GetLanguageBadgeColor(lang string) lipgloss.Color {
	switch strings.ToLower(lang) {
	case "go", "golang":
		return lipgloss.Color("#00E5FF")
	case "py", "python":
		return lipgloss.Color("#FFD438")
	case "js", "javascript":
		return lipgloss.Color("#FFE600")
	case "ts", "typescript":
		return lipgloss.Color("#61AFEF")
	case "rs", "rust":
		return lipgloss.Color("#FF6E4A")
	case "c", "cpp", "c++", "cxx":
		return lipgloss.Color("#82B1FF")
	case "java", "kt", "kotlin":
		return lipgloss.Color("#FFA726")
	case "sh", "bash", "zsh", "shell":
		return lipgloss.Color("#50FA7B")
	case "json", "yaml", "yml", "toml":
		return lipgloss.Color("#FF79C6") // Warm pink matching deepmd
	case "html", "xml", "svg":
		return lipgloss.Color("#FF7043")
	case "css", "scss":
		return lipgloss.Color("#40C4FF")
	case "sql", "psql", "mysql":
		return lipgloss.Color("#FFB74D")
	case "hex", "binary":
		return lipgloss.Color("#FFA726")
	case "text", "raw", "string":
		return lipgloss.Color("#82B1FF")
	default:
		return lipgloss.Color("#BD93F9")
	}
}

// RenderBoxedContent formats payload content inside a borderless card with a solid dark gray
// background, language badge, and Chroma syntax highlighting (identical to deepmd).
func RenderBoxedContent(code, lang string, width int) string {
	if width < 30 {
		width = 80
	}

	// Margin and box sizing with safety padding to prevent terminal edge wrap
	margin := " "
	if width >= 50 {
		margin = "  "
	}

	boxW := width - (len(margin) * 2) - 1
	if boxW < 24 {
		boxW = 24
	}
	innerW := boxW - 4
	if innerW < 12 {
		innerW = 12
	}

	cleanLang := strings.TrimSpace(strings.ToLower(lang))

	// Clean, modern dark gray background (#262626 / RGB(38,38,38))
	bgEscape := "\x1b[48;2;38;38;38m"
	const bgReset = "\x1b[0m"

	// Strip any existing ANSI escape sequences so Chroma parses pure source tokens
	code = xansi.Strip(code)

	// Expand tabs to 4 spaces for consistent terminal alignment and trim trailing newlines
	code = strings.TrimRight(code, "\r\n")
	code = strings.ReplaceAll(code, "\t", "    ")

	// Chroma syntax highlighting
	lexer := lexers.Get(cleanLang)
	if lexer == nil && cleanLang != "" {
		lexer = lexers.Match(cleanLang)
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	themeStyle := styles.Get("dracula")
	if themeStyle == nil {
		themeStyle = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	var buf bytes.Buffer
	iterator, err := lexer.Tokenise(nil, code)
	if err == nil {
		_ = formatter.Format(&buf, themeStyle, iterator)
	} else {
		buf.WriteString(code)
	}
	highlighted := buf.String()

	rawLines := strings.Split(highlighted, "\n")
	for len(rawLines) > 0 && strings.TrimSpace(xansi.Strip(rawLines[len(rawLines)-1])) == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}

	var sb strings.Builder

	// Top padding line on gray background (clean content card without redundant type label)
	topLine := bgEscape + strings.Repeat(" ", boxW) + bgReset
	sb.WriteString(margin + topLine + "\n")

	// Content lines: code lines rendered on gray background with wrapping
	if len(rawLines) == 0 {
		emptyLine := bgEscape + strings.Repeat(" ", boxW) + bgReset
		sb.WriteString(margin + emptyLine + "\n")
	} else {
		for _, line := range rawLines {
			w := lipgloss.Width(line)
			if w <= innerW {
				styledLine := strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgEscape)
				pad := strings.Repeat(" ", innerW-w)
				sb.WriteString(margin + bgEscape + "  " + styledLine + pad + "  " + bgReset + "\n")
			} else {
				wrapped := wrap.String(line, innerW)
				sublines := strings.Split(wrapped, "\n")
				for _, sub := range sublines {
					subStyled := strings.ReplaceAll(sub, "\x1b[0m", "\x1b[0m"+bgEscape)
					subW := lipgloss.Width(sub)
					subPad := strings.Repeat(" ", max(0, innerW-subW))
					sb.WriteString(margin + bgEscape + "  " + subStyled + subPad + "  " + bgReset + "\n")
				}
			}
		}
	}

	// Bottom padding line on gray background
	bottomLine := bgEscape + strings.Repeat(" ", boxW) + bgReset
	sb.WriteString(margin + bottomLine + "\n")

	return sb.String()
}

// DetectStringLanguage inspects a string payload to guess its code language badge.
func DetectStringLanguage(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return "xml"
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "INSERT ") || strings.HasPrefix(upper, "CREATE ") {
		return "sql"
	}
	if strings.HasPrefix(trimmed, "---") {
		return "yaml"
	}
	// Multiline YAML check
	if strings.Contains(trimmed, "\n") && (strings.Contains(trimmed, ": ") || strings.HasPrefix(trimmed, "- ")) {
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(trimmed), &node); err == nil && len(node.Content) > 0 {
			if node.Content[0].Kind == yaml.MappingNode || node.Content[0].Kind == yaml.SequenceNode {
				return "yaml"
			}
		}
	}
	return "text"
}

// RenderInspectContent renders the decoded payload, applying the boxed card background for strings/json/binary
// or returning formatted tables for collections.
func RenderInspectContent(decoded *DecodedResult, width int, compact bool) string {
	if decoded == nil {
		return ""
	}

	if compact {
		return decoded.Formatted
	}

	switch decoded.ValueType {
	case TypeJSON:
		return RenderBoxedContent(decoded.Formatted, "json", width)
	case TypeString:
		lang := DetectStringLanguage(decoded.Formatted)
		return RenderBoxedContent(decoded.Formatted, lang, width)
	case TypeBinary:
		return RenderBoxedContent(decoded.Formatted, "hex", width)
	default:
		// Collections (HASH, LIST, SET, ZSET, STREAM) are structured tables
		return decoded.Formatted
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
