package tui

import (
	"context"
	"fmt"
	"os"

	fzf "github.com/junegunn/fzf/src"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
)

// RunFZFExplorer launches an integrated full-screen interactive FZF key explorer
// with warm-themed styling, non-blocking SCAN streaming, and live key preview.
// Returns:
// - (key, found, err): selected key, whether any keys were found, and error if any.
func RunFZFExplorer(ctx context.Context, cl *client.Client, targetCtx *config.Context, pattern string) (string, bool, error) {
	if pattern == "" {
		pattern = "*"
	}

	// 1. Initial scan to check if there are keys in the database
	scanRes, err := cl.ScanKeys(ctx, pattern, 0, 200)
	if err != nil {
		return "", false, fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(scanRes.Keys) == 0 && scanRes.Cursor == 0 {
		return "", false, nil
	}

	inputChan := make(chan string, 500)

	// Feed first batch into channel
	for _, k := range scanRes.Keys {
		inputChan <- k.Key
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
					inputChan <- k.Key
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

	previewCmd := fmt.Sprintf("%s inspect --context %s {1}", selfBin, targetCtx.Name)
	supplier := targetCtx.Supplier
	if supplier == "" {
		supplier = "Redis"
	}

	fzfArgs := []string{
		"--reverse",
		"--height=100%",
		"--border=rounded",
		fmt.Sprintf("--border-label= 󰌠 REDIS EXPLORER · %s (DB %d) ", targetCtx.Name, targetCtx.DB),
		"--prompt=🔍 Keys > ",
		"--pointer=▶",
		"--marker=✓",
		"--info=inline",
		fmt.Sprintf("--header=Context: %s (%s) | [Enter] inspect key | [Esc/Ctrl-C] quit", targetCtx.Name, supplier),
		"--preview-window=right:60%:wrap:border-rounded",
		"--preview-label= 󰋽 KEY INSPECTION PREVIEW ",
		"--preview=" + previewCmd,
		"--color=fg:#ECEFF4,bg:-1,hl:#FF5722,fg+:#FFFFFF,bg+:#2E3440,hl+:#FF7043",
		"--color=info:#FFA726,prompt:#FF5722,pointer:#FF5722,marker:#4CAF50,spinner:#FF5722,header:#FF8A65",
		"--color=border:#E0382B,label:#FF7043,preview-border:#E0382B,preview-label:#FF7043",
	}

	opts, err := fzf.ParseOptions(true, fzfArgs)
	if err != nil {
		return "", true, fmt.Errorf("failed to parse fzf options: %w", err)
	}

	opts.Input = inputChan

	var selectedKey string
	opts.Printer = func(str string) {
		selectedKey = str
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
