package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/format"
	"redis-cli-tool/pkg/safety"
)

var (
	getFullFlag bool
	setTTLFlag  time.Duration
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get string value of a key",
	Long:  `Retrieve the string value of a key with automatic JSON pretty-printing, hex dump for binary data, and a 256KB preview safety cap.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set string value of a key",
	Long:  `Set the string value of a key with optional TTL expiration.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runSet,
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)

	getCmd.Flags().BoolVar(&getFullFlag, "full", false, "Display full payload bypassing 256KB preview cap")
	setCmd.Flags().DurationVar(&setTTLFlag, "ttl", 0, "TTL expiration duration (e.g. 60s, 10m, 2h)")
}

func runGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	cl, _, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	var decoded *format.DecodedResult

	if !getFullFlag {
		strLen, err := cl.StrLen(ctx, key).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return safety.NewSafetyError(safety.ExitKeyNotFound, fmt.Sprintf("key %q not found", key))
			}
			return fmt.Errorf("failed to get key %q length: %w", key, err)
		}
		if strLen > format.MaxPreviewBytes {
			raw, err := cl.GetRange(ctx, key, 0, format.MaxPreviewBytes-1).Bytes()
			if err != nil {
				return fmt.Errorf("failed to get key %q preview: %w", key, err)
			}
			decoded = format.DecodeStringPayloadWithTotal(raw, strLen, false)
		} else {
			raw, err := cl.Get(ctx, key).Bytes()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					return safety.NewSafetyError(safety.ExitKeyNotFound, fmt.Sprintf("key %q not found", key))
				}
				return fmt.Errorf("failed to get key %q: %w", key, err)
			}
			decoded = format.DecodeStringPayload(raw, true)
		}
	} else {
		raw, err := cl.Get(ctx, key).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return safety.NewSafetyError(safety.ExitKeyNotFound, fmt.Sprintf("key %q not found", key))
			}
			return fmt.Errorf("failed to get key %q: %w", key, err)
		}
		decoded = format.DecodeStringPayload(raw, true)
	}

	if jsonFlag || formatFlag == "json" {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"key":        key,
			"value":      decoded.Formatted,
			"value_type": decoded.ValueType,
			"raw_bytes":  decoded.RawBytes,
			"truncated":  decoded.Truncated,
		}, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), format.DisarmANSI(decoded.Formatted))
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cl, _, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	if err := cl.Set(ctx, key, value, setTTLFlag).Err(); err != nil {
		return fmt.Errorf("failed to set key %q: %w", key, err)
	}

	if jsonFlag || formatFlag == "json" {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"success": true,
			"key":     key,
			"status":  "OK",
		}, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "OK")
	return nil
}
