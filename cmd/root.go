package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/safety"
)

var (
	contextFlag string
	formatFlag  string
	jsonFlag    bool
	compactFlag bool
	fieldsFlag  string
	forceFlag   bool
	timeoutFlag int
)

var rootCmd = &cobra.Command{
	Use:           "redis",
	Short:         "Redis CLI Tool",
	Long:          `A modern Redis CLI tool with multi-context support.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if safetyErr, ok := err.(*safety.SafetyError); ok {
			if jsonFlag || formatFlag == "json" {
				safetyErr.EmitJSON(os.Stderr)
			} else {
				fmt.Fprintln(os.Stderr, safetyErr.Message)
			}
			os.Exit(safetyErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ResetFlags restores default flag values (useful for tests)
func ResetFlags() {
	contextFlag = ""
	formatFlag = "table"
	jsonFlag = false
	compactFlag = false
	fieldsFlag = ""
	forceFlag = false
	timeoutFlag = 10
	dryRunFlag = false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&contextFlag, "context", "", "Context to use (overrides current-context)")
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in JSON format (shortcut for --format=json)")
	rootCmd.PersistentFlags().BoolVar(&compactFlag, "compact", false, "Compact output (no color/formatting)")
	rootCmd.PersistentFlags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to display")
	rootCmd.PersistentFlags().BoolVar(&forceFlag, "force", false, "Force operation without prompt")
	rootCmd.PersistentFlags().IntVar(&timeoutFlag, "timeout", 10, "Command timeout in seconds")
}
