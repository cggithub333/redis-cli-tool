package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/format"
	"redis-cli-tool/pkg/safety"
)

var inspectFullFlag bool

var inspectCmd = &cobra.Command{
	Use:   "inspect <key>",
	Short: "Inspect a key's metadata, encoding, and safely preview its value",
	Long:  `Display a comprehensive inspection card for any Redis key with automatic JSON detection, hex dump for binary data, bounded collection sampling (100 items), and 256KB memory safety cap.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().BoolVar(&inspectFullFlag, "full", false, "Display full payload bypassing 256KB preview safety cap")
}

type KeyInspectionResult struct {
	Key         string                 `json:"key"`
	Type        string                 `json:"type"`
	TTL         time.Duration          `json:"ttl"`
	TTLStr      string                 `json:"ttl_str"`
	MemoryBytes int64                  `json:"memory_bytes"`
	Encoding    string                 `json:"encoding"`
	Decoded     *format.DecodedResult  `json:"decoded"`
}

func runInspect(cmd *cobra.Command, args []string) error {
	key := args[0]
	cl, targetCtx, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	keyType, err := cl.Type(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to query key type: %w", err)
	}
	if keyType == "none" {
		return safety.NewSafetyError(safety.ExitKeyNotFound, fmt.Sprintf("key %q does not exist in context %q (DB %d)", key, targetCtx.Name, targetCtx.DB))
	}

	ttl, _ := cl.TTL(ctx, key).Result()
	memory, _ := cl.MemoryUsage(ctx, key).Result()
	encoding, _ := cl.ObjectEncoding(ctx, key).Result()

	decoded, err := format.DecodeRedisKey(ctx, cl, key, keyType, inspectFullFlag)
	if err != nil {
		return fmt.Errorf("failed to decode key %q: %w", key, err)
	}

	result := KeyInspectionResult{
		Key:         key,
		Type:        keyType,
		TTL:         ttl,
		TTLStr:      formatTTL(ttl),
		MemoryBytes: memory,
		Encoding:    encoding,
		Decoded:     decoded,
	}

	if jsonFlag || formatFlag == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal inspection JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1)

	headerText := fmt.Sprintf("KEY: %s\nTYPE: %s   TTL: %s   MEMORY: %s   ENCODING: %s",
		format.ActiveStyle.Render(key),
		format.HealthyStyle.Render(strings.ToUpper(keyType)),
		result.TTLStr,
		formatBytes(memory),
		encoding,
	)

	fmt.Fprintln(cmd.OutOrStdout(), cardStyle.Render(headerText))
	fmt.Fprintln(cmd.OutOrStdout(), "\nVALUE PREVIEW:")
	fmt.Fprintln(cmd.OutOrStdout(), decoded.Formatted)

	return nil
}
