package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
	"redis-cli-tool/pkg/format"
)

// getActiveClient resolves the target context and constructs a connected *client.Client
func getActiveClient() (*client.Client, config.Context, error) {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, config.Context{}, fmt.Errorf("failed to load context configuration: %w", err)
	}

	targetCtx, err := config.ResolveContext(cfg, contextFlag)
	if err != nil {
		return nil, config.Context{}, err
	}

	cl, err := client.NewClient(&targetCtx)
	if err != nil {
		return nil, targetCtx, fmt.Errorf("failed to initialize Redis client [%s:%d]: %w", targetCtx.Host, targetCtx.Port, err)
	}

	return cl, targetCtx, nil
}

// formatBytes delegates to the canonical format.FormatBytes implementation
func formatBytes(bytes int64) string {
	return format.FormatBytes(bytes)
}

// isWriterTerminal checks if the writer points to an interactive terminal
func isWriterTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

// emitJSON writes JSON to the output stream, colorizing with lipgloss if stdout is a TTY and not compact
func emitJSON(cmd *cobra.Command, rawJSON string) {
	w := cmd.OutOrStdout()
	if !compactFlag && isWriterTerminal(w) {
		fmt.Fprintln(w, format.ColorizeJSON(rawJSON))
	} else {
		fmt.Fprintln(w, rawJSON)
	}
}
