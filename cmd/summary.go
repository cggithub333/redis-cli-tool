package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"redis-cli-tool/pkg/format"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Compact Redis health snapshot (< 500 bytes JSON)",
	Long:  `Retrieve a high-density, low-latency health summary of the active Redis instance capturing version, role, memory, clients, keys, hit rate, and ops/sec.`,
	RunE:  runSummary,
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}

type ServerSummary struct {
	Context    string  `json:"context"`
	Version    string  `json:"version"`
	Role       string  `json:"role"`
	UptimeSec  int64   `json:"uptime_sec"`
	Uptime     string  `json:"uptime"`
	Clients    int64   `json:"clients"`
	Memory     string  `json:"memory"`
	TotalKeys  int64   `json:"keys"`
	HitRatePct float64 `json:"hit_rate_pct"`
	OpsPerSec  int64   `json:"ops_per_sec"`
}

func parseInfoMap(infoStr string) map[string]string {
	m := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(infoStr))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}

func runSummary(cmd *cobra.Command, args []string) error {
	cl, targetCtx, err := getActiveClient()
	if err != nil {
		return err
	}
	defer cl.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(timeoutFlag)*time.Second)
	defer cancel()

	infoStr, err := cl.Info(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to retrieve INFO from Redis: %w", err)
	}

	info := parseInfoMap(infoStr)
	dbSize, _ := cl.DBSize(ctx).Result()

	uptimeSec, _ := strconv.ParseInt(info["uptime_in_seconds"], 10, 64)
	clients, _ := strconv.ParseInt(info["connected_clients"], 10, 64)
	ops, _ := strconv.ParseInt(info["instantaneous_ops_per_sec"], 10, 64)

	hits, _ := strconv.ParseFloat(info["keyspace_hits"], 64)
	misses, _ := strconv.ParseFloat(info["keyspace_misses"], 64)
	hitRate := 0.0
	if hits+misses > 0 {
		hitRate = (hits / (hits + misses)) * 100.0
	}

	summary := ServerSummary{
		Context:    targetCtx.Name,
		Version:    info["redis_version"],
		Role:       info["role"],
		UptimeSec:  uptimeSec,
		Uptime:     formatTTL(time.Duration(uptimeSec) * time.Second),
		Clients:    clients,
		Memory:     info["used_memory_human"],
		TotalKeys:  dbSize,
		HitRatePct: hitRate,
		OpsPerSec:  ops,
	}

	if jsonFlag || formatFlag == "json" {
		var data []byte
		if compactFlag {
			data, _ = json.Marshal(summary)
		} else {
			data, _ = json.MarshalIndent(summary, "", "  ")
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(0, 1)

	content := fmt.Sprintf(
		"REDIS SERVER HEALTH: %s\n\nVersion: %-10s  Role: %-10s  Uptime: %s\nClients: %-10d  Keys: %-10d  Memory: %s\nHit Rate: %-9.1f%% Ops/sec: %d",
		format.ActiveStyle.Render(targetCtx.Name),
		format.HealthyStyle.Render(summary.Version),
		summary.Role,
		summary.Uptime,
		summary.Clients,
		summary.TotalKeys,
		summary.Memory,
		summary.HitRatePct,
		summary.OpsPerSec,
	)

	fmt.Fprintln(cmd.OutOrStdout(), cardStyle.Render(content))
	return nil
}
