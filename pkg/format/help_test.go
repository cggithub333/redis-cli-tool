package format

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRenderHelp(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:   "redis",
		Short: "Redis CLI Tool",
		Long:  "A modern Redis CLI tool with multi-context support.",
	}
	subCmd := &cobra.Command{
		Use:   "context",
		Short: "Manage Redis connection contexts",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	rootCmd.AddCommand(subCmd)

	rootCmd.PersistentFlags().String("context", "", "Context to use")
	rootCmd.Flags().BoolP("help", "h", false, "Help for redis")

	out := RenderHelp(rootCmd)
	if !strings.Contains(out, "REDIS") {
		t.Errorf("expected REDIS in help output, got:\n%s", out)
	}
	if !strings.Contains(out, "USAGE") {
		t.Errorf("expected USAGE in help output, got:\n%s", out)
	}
	if !strings.Contains(out, "COMMANDS") {
		t.Errorf("expected COMMANDS in help output, got:\n%s", out)
	}
	if !strings.Contains(out, "context") {
		t.Errorf("expected context command in help output, got:\n%s", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("expected FLAGS in help output, got:\n%s", out)
	}

	subOut := RenderHelp(subCmd)
	if !strings.Contains(subOut, "REDIS CONTEXT") {
		t.Errorf("expected REDIS CONTEXT in subCmd help output, got:\n%s", subOut)
	}
	if !strings.Contains(subOut, "GLOBAL FLAGS") {
		t.Errorf("expected GLOBAL FLAGS in subCmd help output, got:\n%s", subOut)
	}
}
