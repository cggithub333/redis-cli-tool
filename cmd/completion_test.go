package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCommands(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// 1. Zsh completion generation
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"completion", "zsh"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to generate zsh completion: %v", err)
	}
	if !strings.Contains(buf.String(), "compdef _redis redis") {
		t.Errorf("expected compdef _redis redis in zsh completion output")
	}

	// 2. Bash completion generation
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"completion", "bash"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to generate bash completion: %v", err)
	}
	if !strings.Contains(buf.String(), "_redis") {
		t.Errorf("expected _redis in bash completion output")
	}

	// 3. Fish completion generation
	ResetFlags()
	buf.Reset()
	rootCmd.SetArgs([]string{"completion", "fish"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to generate fish completion: %v", err)
	}
	if !strings.Contains(buf.String(), "complete -c redis") {
		t.Errorf("expected complete -c redis in fish completion output")
	}
}
