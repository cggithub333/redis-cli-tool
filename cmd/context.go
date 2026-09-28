package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"redis-cli-tool/pkg/client"
	"redis-cli-tool/pkg/config"
	"redis-cli-tool/pkg/format"
)

var (
	createHostFlag     string
	createPortFlag     int
	createDBFlag       int
	createUsernameFlag string
	createPasswordFlag string
	createTLSFlag      bool
	exportSecretsFlag  bool
)

var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Manage Redis connection contexts",
	Long:    `Create, switch, list, delete, export, and import Redis connection contexts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var contextLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all configured contexts with connection health status",
	RunE:  runContextLs,
}

var contextUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch active context",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextUse,
}

var contextCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print active context name",
	RunE:  runContextCurrent,
}

var contextCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new context profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextCreate,
}

var contextDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm"},
	Short:   "Delete a context profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextDelete,
}

var contextExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export context configuration as YAML",
	RunE:  runContextExport,
}

var contextImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import contexts from a YAML file (or '-' for stdin)",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextImport,
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextLsCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextCurrentCmd)
	contextCmd.AddCommand(contextCreateCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	contextCmd.AddCommand(contextExportCmd)
	contextCmd.AddCommand(contextImportCmd)

	contextCreateCmd.Flags().StringVar(&createHostFlag, "host", "127.0.0.1", "Redis host")
	contextCreateCmd.Flags().IntVar(&createPortFlag, "port", 6379, "Redis port")
	contextCreateCmd.Flags().IntVar(&createDBFlag, "db", 0, "Redis DB index")
	contextCreateCmd.Flags().StringVar(&createUsernameFlag, "username", "", "Redis username (ACL)")
	contextCreateCmd.Flags().StringVar(&createPasswordFlag, "password", "", "Redis password")
	contextCreateCmd.Flags().BoolVar(&createTLSFlag, "tls", false, "Enable TLS connection")

	contextExportCmd.Flags().BoolVar(&exportSecretsFlag, "include-secrets", false, "Include plain passwords without masking")
}

type contextStatusResult struct {
	Active    bool          `json:"active"`
	Name      string        `json:"name"`
	Host      string        `json:"host"`
	Port      int           `json:"port"`
	DB        int           `json:"db"`
	TLS       bool          `json:"tls"`
	Status    string        `json:"status"`
	Latency   time.Duration `json:"-"`
	LatencyMS float64       `json:"latency_ms"`
}

func runContextLs(cmd *cobra.Command, args []string) error {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	if len(cfg.Contexts) == 0 {
		if jsonFlag || formatFlag == "json" {
			cmd.Println("[]")
			return nil
		}
		cmd.Println("No contexts configured. Create one with: redis context create <name>")
		return nil
	}

	// Active context determined by 3-tier precedence
	activeContextName := contextFlag
	if activeContextName == "" {
		activeContextName = os.Getenv("REDIS_CONTEXT")
	}
	if activeContextName == "" {
		activeContextName = cfg.CurrentContext
	}

	results := make([]contextStatusResult, len(cfg.Contexts))
	var wg sync.WaitGroup

	for i, c := range cfg.Contexts {
		wg.Add(1)
		go func(idx int, target config.Context) {
			defer wg.Done()
			res := contextStatusResult{
				Active: target.Name == activeContextName,
				Name:   target.Name,
				Host:   target.Host,
				Port:   target.Port,
				DB:     target.DB,
				TLS:    target.TLS,
				Status: "UNREACHABLE",
			}

			cl, err := client.NewClient(&target)
			if err == nil {
				defer cl.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()

				lat, pingErr := cl.Ping(ctx)
				if pingErr == nil {
					res.Status = "HEALTHY"
					res.Latency = lat
					res.LatencyMS = float64(lat.Microseconds()) / 1000.0
				}
			}
			results[idx] = res
		}(i, c)
	}

	wg.Wait()

	if jsonFlag || formatFlag == "json" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal contexts JSON: %w", err)
		}
		cmd.Println(string(data))
		return nil
	}

	headers := []string{"ACTIVE", "NAME", "ENDPOINT", "DB", "STATUS", "LATENCY"}
	rows := make([][]string, len(results))
	for i, res := range results {
		activeMark := " "
		if res.Active {
			activeMark = "*"
		}

		endpoint := fmt.Sprintf("%s:%d", res.Host, res.Port)
		if res.TLS {
			endpoint += " (tls)"
		}

		latencyStr := "-"
		if res.Status == "HEALTHY" {
			latencyStr = fmt.Sprintf("%.2fms", res.LatencyMS)
		}

		rows[i] = []string{
			activeMark,
			res.Name,
			endpoint,
			fmt.Sprintf("%d", res.DB),
			format.StatusBadge(res.Status),
			latencyStr,
		}
	}

	cmd.Println(format.RenderTable(headers, rows))
	return nil
}

func runContextUse(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	if _, exists := cfg.GetContext(name); !exists {
		return fmt.Errorf("context %q not found", name)
	}

	cfg.CurrentContext = name
	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}

	// Concurrency advisory if running in non-TTY environment
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(cmd.ErrOrStderr(), "Advisory: Global context switched to %q. For concurrent scripts/subagents, prefer --context flag or REDIS_CONTEXT env var.\n", name)
	}

	cmd.Printf("Switched to context %q.\n", name)
	return nil
}

func runContextCurrent(cmd *cobra.Command, args []string) error {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	ctx, err := config.ResolveContext(cfg, contextFlag)
	if err != nil {
		return err
	}

	if jsonFlag || formatFlag == "json" {
		data, _ := json.MarshalIndent(map[string]string{"current_context": ctx.Name}, "", "  ")
		cmd.Println(string(data))
		return nil
	}

	cmd.Println(ctx.Name)
	return nil
}

func runContextCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	if _, exists := cfg.GetContext(name); exists && !forceFlag {
		return fmt.Errorf("context %q already exists (use --force to overwrite)", name)
	}

	newCtx := config.Context{
		Name:     name,
		Host:     createHostFlag,
		Port:     createPortFlag,
		DB:       createDBFlag,
		Username: createUsernameFlag,
		Password: createPasswordFlag,
		TLS:      createTLSFlag,
	}

	cfg.SetContext(newCtx)
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = name
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	cmd.Printf("Context %q created.\n", name)
	return nil
}

func runContextDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	if !cfg.DeleteContext(name) {
		return fmt.Errorf("context %q not found", name)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}

	cmd.Printf("Context %q deleted.\n", name)
	return nil
}

func runContextExport(cmd *cobra.Command, args []string) error {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	targetCfg := cfg
	if !exportSecretsFlag {
		targetCfg = config.SanitizeConfig(cfg)
	}

	data, err := yaml.Marshal(targetCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal export config: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), string(data))
	return nil
}

func runContextImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	var data []byte
	var err error

	if filePath == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var importedCfg config.Config
	if err := yaml.Unmarshal(data, &importedCfg); err != nil {
		return fmt.Errorf("failed to unmarshal import config: %w", err)
	}

	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	for _, importedCtx := range importedCfg.Contexts {
		cfg.SetContext(importedCtx)
	}

	if importedCfg.CurrentContext != "" {
		cfg.CurrentContext = importedCfg.CurrentContext
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}

	cmd.Printf("Successfully imported %d context(s).\n", len(importedCfg.Contexts))
	return nil
}
