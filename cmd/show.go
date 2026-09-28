package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/format"
)

var (
	showPatternFlag string
	showCursorFlag  uint64
	showLimitFlag   int64
	showPageFlag    int
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Browse and scan keys with metadata",
	Long:  `Scan keys in the active Redis context without blocking Redis threads, displaying Type, TTL, and Memory usage.`,
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().StringVarP(&showPatternFlag, "pattern", "p", "*", "Pattern to match keys (glob syntax)")
	showCmd.Flags().Uint64Var(&showCursorFlag, "cursor", 0, "Scan cursor to start from")
	showCmd.Flags().Int64VarP(&showLimitFlag, "limit", "l", 50, "Maximum number of keys to return per batch")
	showCmd.Flags().IntVar(&showPageFlag, "page", 1, "Page number (calculated when cursor=0)")
}

func formatTTL(d time.Duration) string {
	if d == -1 {
		return "none"
	}
	if d == -2 {
		return "expired"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func runShow(cmd *cobra.Command, args []string) error {
	cl, targetCtx, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	cursor := showCursorFlag

	// Support client-side page skipping if --page > 1 and cursor is 0
	if showPageFlag > 1 && showCursorFlag == 0 {
		for p := 1; p < showPageFlag; p++ {
			skipRes, err := cl.ScanKeys(ctx, showPatternFlag, cursor, showLimitFlag)
			if err != nil {
				return fmt.Errorf("failed to scan page %d: %w", p, err)
			}
			cursor = skipRes.Cursor
			if cursor == 0 {
				break
			}
		}
	}

	res, err := cl.ScanKeys(ctx, showPatternFlag, cursor, showLimitFlag)
	if err != nil {
		return fmt.Errorf("failed to scan keys in context %q: %w", targetCtx.Name, err)
	}

	for i := range res.Keys {
		res.Keys[i].TTLStr = formatTTL(res.Keys[i].TTL)
	}

	if jsonFlag || formatFlag == "json" {
		format.PurifyStream()
		if fieldsFlag != "" {
			fields := strings.Split(fieldsFlag, ",")
			filteredKeys := make([]map[string]interface{}, len(res.Keys))
			for i, k := range res.Keys {
				filteredKeys[i] = format.FilterFields(k, fields)
			}
			outStr, err := format.SerializeJSON(filteredKeys, compactFlag)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), outStr)
			return nil
		}

		outStr, err := format.SerializeJSON(res, compactFlag)
		if err != nil {
			return fmt.Errorf("failed to marshal scan result: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), outStr)
		return nil
	}

	if len(res.Keys) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No keys matching pattern %q found in context %q (DB %d).\n", showPatternFlag, targetCtx.Name, targetCtx.DB)
		return nil
	}

	headers := []string{"KEY", "TYPE", "TTL", "MEMORY"}
	rows := make([][]string, len(res.Keys))
	for i, k := range res.Keys {
		typeUpper := strings.ToUpper(k.Type)
		typeStyled := typeUpper
		switch typeUpper {
		case "STRING":
			typeStyled = format.HealthyStyle.Render(typeUpper)
		case "HASH":
			typeStyled = format.ActiveStyle.Render(typeUpper)
		case "LIST":
			typeStyled = format.WarningStyle.Render(typeUpper)
		case "SET", "ZSET":
			typeStyled = format.HeaderStyle.Render(typeUpper)
		}

		rows[i] = []string{
			k.Key,
			typeStyled,
			k.TTLStr,
			formatBytes(k.Memory),
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), format.RenderTable(headers, rows))

	if res.Cursor != 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d keys. Next cursor: %d (use --cursor=%d for next batch)\n", len(res.Keys), res.Cursor, res.Cursor)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\nShowing %d keys. (Scan complete, cursor returned to 0)\n", len(res.Keys))
	}

	return nil
}
