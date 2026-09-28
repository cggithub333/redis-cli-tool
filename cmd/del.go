package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/safety"
)

var dryRunFlag bool

var delCmd = &cobra.Command{
	Use:     "del <pattern_or_keys...>",
	Aliases: []string{"unlink", "rm"},
	Short:   "Safely delete keys or patterns with chunked UNLINK",
	Long:    `Delete one or more keys or patterns. Patterns require --force in non-interactive environments and stream deletions in safe chunks of 500 keys using UNLINK. Supports --dry-run.`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    runDel,
}

func init() {
	rootCmd.AddCommand(delCmd)
	delCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Simulate deletion without removing keys, reporting count and memory")
}

func isPattern(s string) bool {
	return strings.ContainsAny(s, "*?[]")
}

func runDel(cmd *cobra.Command, args []string) error {
	cl, targetCtx, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	// Check if any argument is a pattern
	hasPattern := false
	for _, arg := range args {
		if isPattern(arg) {
			hasPattern = true
			break
		}
	}

	if hasPattern && !dryRunFlag {
		desc := fmt.Sprintf("pattern deletion of %s in context %q", strings.Join(args, ", "), targetCtx.Name)
		if err := safety.CheckDestructiveAction(isTTY, forceFlag, desc, cmd.InOrStdin(), cmd.ErrOrStderr()); err != nil {
			return err
		}
	}

	const batchSize = 500
	var totalMatched int64
	var totalUnlinked int64
	var estimatedMemory int64

	for _, target := range args {
		if isPattern(target) {
			var cursor uint64
			for {
				keys, nextCursor, err := cl.Scan(ctx, cursor, target, batchSize).Result()
				if err != nil {
					return fmt.Errorf("failed to scan keys for pattern %q: %w", target, err)
				}

				totalMatched += int64(len(keys))

				if len(keys) > 0 {
					if dryRunFlag {
						pipe := cl.Pipeline()
						memCmds := make([]*struct{ mem int64 }, len(keys))
						_ = memCmds
						for _, k := range keys {
							pipe.MemoryUsage(ctx, k)
						}
						cmders, _ := pipe.Exec(ctx)
						for _, cmder := range cmders {
							if intCmd, ok := cmder.(interface{ Val() int64 }); ok {
								estimatedMemory += intCmd.Val()
							}
						}
					} else {
						unlinked, err := cl.Unlink(ctx, keys...).Result()
						if err != nil {
							return fmt.Errorf("failed to unlink batch: %w", err)
						}
						totalUnlinked += unlinked
					}
				}

				cursor = nextCursor
				if cursor == 0 {
					break
				}
			}
		} else {
			// Single explicit key
			totalMatched++
			if dryRunFlag {
				mem, _ := cl.MemoryUsage(ctx, target).Result()
				estimatedMemory += mem
			} else {
				unlinked, err := cl.Unlink(ctx, target).Result()
				if err != nil {
					return fmt.Errorf("failed to unlink key %q: %w", target, err)
				}
				totalUnlinked += unlinked
			}
		}
	}

	if dryRunFlag {
		if jsonFlag || formatFlag == "json" {
			data, _ := json.MarshalIndent(map[string]interface{}{
				"dry_run":                true,
				"matched_keys":           totalMatched,
				"estimated_memory_bytes": estimatedMemory,
			}, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[DRY-RUN] Matched %d key(s) (~%s memory). 0 keys deleted.\n", totalMatched, formatBytes(estimatedMemory))
		return nil
	}

	if jsonFlag || formatFlag == "json" {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"success":       true,
			"unlinked_keys": totalUnlinked,
			"matched_keys":  totalMatched,
		}, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully unlinked %d key(s).\n", totalUnlinked)
	return nil
}
