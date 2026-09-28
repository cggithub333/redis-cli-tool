package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/safety"
	"redis-cli-tool/pkg/tui"
)

var exploreCmd = &cobra.Command{
	Use:     "explore",
	Aliases: []string{"ui"},
	Short:   "Launch full-screen interactive TUI key explorer",
	Long:    `Launch an interactive full-screen terminal browser with live search filtering, split-pane inspection, and keyboard navigation. Requires an interactive TTY.`,
	RunE:    runExplore,
}

func init() {
	rootCmd.AddCommand(exploreCmd)
}

func runExplore(cmd *cobra.Command, args []string) error {
	// Preflight guardrail: strictly reject non-TTY sessions (CI / agent pipes) with ExitSyntaxError (code 5)
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return safety.NewSafetyError(safety.ExitSyntaxError, "redis explore requires an interactive TTY. Use 'redis show' instead.")
	}

	cl, targetCtx, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	scanRes, err := cl.ScanKeys(ctx, "*", 0, 200)
	if err != nil {
		return fmt.Errorf("failed to scan keys for TUI: %w", err)
	}

	for i := range scanRes.Keys {
		scanRes.Keys[i].TTLStr = formatTTL(scanRes.Keys[i].TTL)
	}

	m := tui.NewModel(cl, targetCtx.Name, scanRes.Keys)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI execution failed: %w", err)
	}

	return nil
}
