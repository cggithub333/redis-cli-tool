package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	fzf "github.com/junegunn/fzf/src"
	"github.com/muesli/termenv"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
)

var (
	fzfTypeString = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E5FF")) // Electric Cyan
	fzfTypeHash   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E040FB")) // Neon Magenta
	fzfTypeList   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E676")) // Bright Green
	fzfTypeSet    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD600")) // Bright Yellow
	fzfTypeZSet   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6E40")) // Warm Orange
	fzfTypeStream = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B388FF")) // Light Purple
	fzfTypeOther  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#B0BEC5")) // Gray

	fzfTTLDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#757575")) // Gray
	fzfTTLNorm   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ECEFF4")) // Light White
	fzfTTLWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA726")) // Amber
	fzfTTLUrgent = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5252")) // Red

	fzfMemStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE")) // Slate Gray
	fzfDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#546E7A")) // Subtle Divider

	fzfNamespaceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8A65"))           // Warm Coral
	fzfKeyNameStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")) // Bright White
	fzfKeyBadgeStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8A65")) // Warm Coral for shortcuts
	fzfDescStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE"))             // Slate for descriptions
	fzfHeaderColsStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8A65")) // Header column names
	fzfHeaderRulerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#455A64"))           // Subtle dark ruler
)

// FormatFZFTypeBadge returns a stylized, fixed-width badge for Redis key types.
func FormatFZFTypeBadge(typ string) string {
	switch strings.ToLower(typ) {
	case "string":
		return fzfTypeString.Render(" STRING ")
	case "hash":
		return fzfTypeHash.Render("  HASH  ")
	case "list":
		return fzfTypeList.Render("  LIST  ")
	case "set":
		return fzfTypeSet.Render("  SET   ")
	case "zset":
		return fzfTypeZSet.Render("  ZSET  ")
	case "stream":
		return fzfTypeStream.Render(" STREAM ")
	default:
		padded := fmt.Sprintf("%-8s", strings.ToUpper(typ))
		if len(padded) > 8 {
			padded = padded[:8]
		}
		return fzfTypeOther.Render(padded)
	}
}

// FormatFZFTTL formats a TTL duration into a stylized, fixed-width column string.
func FormatFZFTTL(ttl time.Duration) string {
	if ttl == -1 {
		return fzfTTLDim.Render(" persist")
	}
	if ttl <= 0 {
		return fzfTTLDim.Render("    none")
	}

	var str string
	if ttl >= time.Hour {
		h := int(ttl.Hours())
		m := int(ttl.Minutes()) % 60
		str = fmt.Sprintf("%dh%02dm", h, m)
	} else if ttl >= time.Minute {
		m := int(ttl.Minutes())
		s := int(ttl.Seconds()) % 60
		str = fmt.Sprintf("%dm%02ds", m, s)
	} else {
		str = fmt.Sprintf("%ds", int(ttl.Seconds()))
	}

	aligned := fmt.Sprintf("%8s", str)
	if ttl < 60*time.Second {
		return fzfTTLUrgent.Render(aligned)
	}
	if ttl < 3600*time.Second {
		return fzfTTLWarn.Render(aligned)
	}
	return fzfTTLNorm.Render(aligned)
}

// FormatFZFMemory formats memory usage into a stylized, fixed-width column string.
func FormatFZFMemory(bytes int64) string {
	var str string
	if bytes < 1024 {
		str = fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		str = fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		str = fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fzfMemStyle.Render(fmt.Sprintf("%8s", str))
}

// FormatFZFKey splits namespaces and highlights key components.
func FormatFZFKey(key string) string {
	if idx := strings.LastIndex(key, ":"); idx != -1 {
		ns := key[:idx+1]
		name := key[idx+1:]
		return fzfNamespaceStyle.Render(ns) + fzfKeyNameStyle.Render(name)
	}
	return fzfKeyNameStyle.Render(key)
}

// FormatFZFItem packages a key metadata into a tab-delimited line for FZF.
// Field 1: raw key string (used by preview & printer)
// Field 2: formatted and stylized line with ANSI colors (displayed to user)
func FormatFZFItem(k client.KeyMetadata) string {
	display := fmt.Sprintf("%s  %s  %s %s%s",
		FormatFZFTypeBadge(k.Type),
		FormatFZFTTL(k.TTL),
		FormatFZFMemory(k.Memory),
		fzfDividerStyle.Render("│ "),
		FormatFZFKey(k.Key),
	)
	return fmt.Sprintf("%s\t%s", k.Key, display)
}

// RunFZFExplorer launches an integrated full-screen interactive FZF key explorer
// with warm-themed styling, non-blocking SCAN streaming, and live key preview.
// Returns:
// - (key, found, err): selected key, whether any keys were found, and error if any.
func RunFZFExplorer(ctx context.Context, cl *client.Client, targetCtx *config.Context, pattern string) (string, bool, error) {
	if pattern == "" {
		pattern = "*"
	}

	// Ensure ANSI color profile is active
	lipgloss.SetColorProfile(termenv.TrueColor)

	// 1. Initial scan to check if there are keys in the database
	scanRes, err := cl.ScanKeys(ctx, pattern, 0, 200)
	if err != nil {
		return "", false, fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(scanRes.Keys) == 0 && scanRes.Cursor == 0 {
		return "", false, nil
	}

	inputChan := make(chan string, 500)

	// Feed pinned table header rows first (header-lines=2) so they appear right below the filter prompt
	headerCols := fzfHeaderColsStyle.Render("  TYPE      TTL       MEMORY │ KEY")
	headerRuler := fzfHeaderRulerStyle.Render("─────────────────────────────┼────────────────────────────────────────────")
	inputChan <- fmt.Sprintf("\t%s", headerCols)
	inputChan <- fmt.Sprintf("\t%s", headerRuler)

	// Feed first batch into channel
	for _, k := range scanRes.Keys {
		inputChan <- FormatFZFItem(k)
	}

	if scanRes.Cursor == 0 {
		close(inputChan)
	} else {
		// Background stream remaining keys asynchronously so fzf opens immediately
		go func(initialCursor uint64) {
			defer close(inputChan)
			cursor := initialCursor
			totalScanned := len(scanRes.Keys)
			const maxKeys = 20000 // Guardrail against client memory exhaustion

			for cursor != 0 && totalScanned < maxKeys {
				select {
				case <-ctx.Done():
					return
				default:
				}

				res, err := cl.ScanKeys(ctx, pattern, cursor, 200)
				if err != nil || len(res.Keys) == 0 {
					break
				}
				for _, k := range res.Keys {
					inputChan <- FormatFZFItem(k)
					totalScanned++
					if totalScanned >= maxKeys {
						break
					}
				}
				cursor = res.Cursor
			}
		}(scanRes.Cursor)
	}

	// 2. Discover self executable path for the preview command
	selfBin, err := os.Executable()
	if err != nil || selfBin == "" {
		selfBin = "redis"
	}

	sessionID := fmt.Sprintf("%d_%d", os.Getpid(), time.Now().UnixNano())
	_, cleanup := InitResizeSession(sessionID)
	defer cleanup()

	previewCmd := fmt.Sprintf("CLICOLOR_FORCE=1 %s inspect --context %s {1}", selfBin, targetCtx.Name)
	supplier := targetCtx.Supplier
	if supplier == "" {
		supplier = "Redis"
	}

	headerText := StandardHeaderText()

	borderLabel := fmt.Sprintf("  REDIS EXPLORER · %s (DB %d) · Pattern: %s ", targetCtx.Name, targetCtx.DB, pattern)

	resizeCmdStr := func(action string) string {
		return fmt.Sprintf("transform(%s __resize %s %s)", selfBin, sessionID, action)
	}

	fzfArgs := []string{
		"--ansi",
		"--delimiter=\t",
		"--with-nth=2",
		"--layout=reverse-list",
		"--header-lines=2",
		"--height=100%",
		"--border=rounded",
		"--border-label=" + borderLabel,
		"--border-label-pos=0:top",
		"--prompt=󰍉 Filter Keys > ",
		"--pointer=▶",
		"--marker=✓",
		"--info=inline",
		"--no-separator",
		"--header=" + headerText,
		"--header-border=bottom",
		"--preview-window=right:55%:wrap:border-rounded",
		"--preview-label=  KEY INSPECTION PREVIEW ",
		"--preview=" + previewCmd,
		"--bind=ctrl-d:preview-page-down,ctrl-u:preview-page-up,ctrl-/:toggle-preview",
		"--bind=start:unbind(left,right)",
		"--bind=alt-q:" + resizeCmdStr("toggle"),
		"--bind=alt-r:" + resizeCmdStr("toggle"),
		"--bind=esc:" + resizeCmdStr("esc"),
		"--bind=left:" + resizeCmdStr("left"),
		"--bind=right:" + resizeCmdStr("right"),
		"--bind=alt-left:" + resizeCmdStr("left"),
		"--bind=alt-right:" + resizeCmdStr("right"),
		"--bind=shift-left:" + resizeCmdStr("left"),
		"--bind=shift-right:" + resizeCmdStr("right"),
		"--bind=alt-h:" + resizeCmdStr("left"),
		"--bind=alt-l:" + resizeCmdStr("right"),
		"--color=fg:#ECEFF4,bg:-1,hl:#FF5722,fg+:#FFFFFF,bg+:#2E3440,hl+:#FF7043",
		"--color=info:#FFA726,prompt:#FF5722,pointer:#FF5722,marker:#4CAF50,spinner:#FF5722,header:#FF8A65",
		"--color=border:#37474F,label:#FF7043,preview-border:#37474F,preview-label:#FF7043",
	}

	opts, err := fzf.ParseOptions(true, fzfArgs)
	if err != nil {
		return "", true, fmt.Errorf("failed to parse fzf options: %w", err)
	}

	opts.Input = inputChan

	var selectedKey string
	opts.Printer = func(str string) {
		parts := strings.SplitN(str, "\t", 2)
		if len(parts) > 0 {
			selectedKey = parts[0]
		}
	}

	exitCode, err := fzf.Run(opts)
	if err != nil {
		return "", true, fmt.Errorf("fzf execution failed: %w", err)
	}

	if exitCode != 0 {
		// User aborted with Esc or Ctrl-C
		return "", true, nil
	}

	return selectedKey, true, nil
}
