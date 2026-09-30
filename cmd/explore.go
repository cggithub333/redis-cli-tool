package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/safety"
	"redis-cli-tool/pkg/tui"
)

var explorePatternFlag string

var exploreCmd = &cobra.Command{
	Use:     "explore [pattern]",
	Aliases: []string{"ui"},
	Short:   "Launch full-screen interactive FZF key explorer with live preview",
	Long:    `Launch an interactive full-screen fuzzy finder with live search, split-pane inspection preview, and keyboard navigation powered by an integrated FZF engine. Requires an interactive TTY.`,
	RunE:    runExplore,
}

var resizeCmd = &cobra.Command{
	Use:    "__resize [session-id] [action]",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.HandleResize(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(exploreCmd)
	rootCmd.AddCommand(resizeCmd)
	exploreCmd.Flags().StringVarP(&explorePatternFlag, "pattern", "p", "*", "Pattern to match keys (glob syntax)")
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

	pattern := explorePatternFlag
	if len(args) > 0 {
		pattern = args[0]
	}

	selectedKey, found, err := tui.RunFZFExplorer(ctx, cl, &targetCtx, pattern)
	if err != nil {
		return err
	}

	if !found {
		fmt.Fprintf(cmd.OutOrStdout(), "No keys matching pattern %q found in context %q (DB %d).\n", pattern, targetCtx.Name, targetCtx.DB)
		return nil
	}

	if selectedKey != "" {
		return runInspect(cmd, []string{selectedKey})
	}

	return nil
}
