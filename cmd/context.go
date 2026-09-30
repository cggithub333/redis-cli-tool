package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/redis/go-redis/v9"
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
	createURIFlag      string
	createSupplierFlag string
	renameSupplierFlag string
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
	Use:   "create [name]",
	Short: "Create a new context profile",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runContextCreate,
}

var contextRenameCmd = &cobra.Command{
	Use:     "rename <old-name> <new-name>",
	Aliases: []string{"mv"},
	Short:   "Rename an existing connection context",
	Args:    cobra.ExactArgs(2),
	RunE:    runContextRename,
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
	contextCmd.AddCommand(contextRenameCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	contextCmd.AddCommand(contextExportCmd)
	contextCmd.AddCommand(contextImportCmd)

	contextCreateCmd.Flags().StringVar(&createHostFlag, "host", "127.0.0.1", "Redis host")
	contextCreateCmd.Flags().IntVar(&createPortFlag, "port", 6379, "Redis port")
	contextCreateCmd.Flags().IntVar(&createDBFlag, "db", 0, "Redis DB index")
	contextCreateCmd.Flags().StringVar(&createUsernameFlag, "username", "", "Redis username (ACL)")
	contextCreateCmd.Flags().StringVar(&createPasswordFlag, "password", "", "Redis password")
	contextCreateCmd.Flags().BoolVar(&createTLSFlag, "tls", false, "Enable TLS connection")
	contextCreateCmd.Flags().StringVar(&createURIFlag, "uri", "", "Redis connection URI (e.g. redis:// or rediss://)")
	contextCreateCmd.Flags().StringVar(&createURIFlag, "url", "", "Redis connection URL (alias for --uri)")
	contextCreateCmd.Flags().StringVar(&createSupplierFlag, "supplier", "", "Redis supplier/provider (e.g. 'Layerbase', 'Redis Official', 'Local container')")

	contextRenameCmd.Flags().StringVar(&renameSupplierFlag, "supplier", "", "Update supplier for the context")

	contextExportCmd.Flags().BoolVar(&exportSecretsFlag, "include-secrets", false, "Include plain passwords without masking")

	contextUseCmd.ValidArgsFunction = CompleteContextNames
	contextDeleteCmd.ValidArgsFunction = CompleteContextNames
	contextRenameCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return CompleteContextNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	contextCreateCmd.RegisterFlagCompletionFunc("supplier", CompleteSuppliers)
	contextRenameCmd.RegisterFlagCompletionFunc("supplier", CompleteSuppliers)
}

// CompleteContextNames dynamically lists configured context names with supplier and endpoint descriptions.
func CompleteContextNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load(config.DefaultConfigPath())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var results []string
	for _, c := range cfg.Contexts {
		desc := c.Supplier
		if desc == "" {
			desc = fmt.Sprintf("%s:%d", c.Host, c.Port)
		} else {
			desc = fmt.Sprintf("%s (%s:%d)", desc, c.Host, c.Port)
		}
		results = append(results, fmt.Sprintf("%s\t%s", c.Name, desc))
	}
	return results, cobra.ShellCompDirectiveNoFileComp
}

// CompleteSuppliers provides suggestions for common Redis hosting providers.
func CompleteSuppliers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"Local container\tLocal Docker or Podman container",
		"Layerbase\tLayerbase Cloud Redis",
		"Redis Official\tRedis Cloud / Enterprise",
		"Upstash\tUpstash Serverless Redis",
		"AWS ElastiCache\tAWS ElastiCache Redis cluster",
		"Aiven\tAiven Managed Redis",
	}, cobra.ShellCompDirectiveNoFileComp
}

type contextStatusResult struct {
	Active    bool          `json:"active"`
	Name      string        `json:"name"`
	Supplier  string        `json:"supplier"`
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
			fmt.Fprintln(cmd.OutOrStdout(), "[]")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No contexts configured. Create one with: redis context create <name>")
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
				Active:   target.Name == activeContextName,
				Name:     target.Name,
				Supplier: config.DetectSupplier(target.Host, target.Supplier),
				Host:     target.Host,
				Port:     target.Port,
				DB:       target.DB,
				TLS:      target.TLS,
				Status:   "UNREACHABLE",
			}

			cl, err := client.NewClient(&target)
			if err == nil {
				defer cl.Close()
				pingTimeout := 2 * time.Second
				if timeoutFlag > 0 && time.Duration(timeoutFlag)*time.Second > pingTimeout {
					pingTimeout = time.Duration(timeoutFlag) * time.Second
				}
				ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
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
		emitJSON(cmd, string(data))
		return nil
	}

	headers := []string{"ACTIVE", "NAME", "SUPPLIER", "ENDPOINT", "DB", "STATUS", "LATENCY"}
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
			res.Supplier,
			endpoint,
			fmt.Sprintf("%d", res.DB),
			format.StatusBadge(res.Status),
			latencyStr,
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), format.RenderTable(headers, rows))
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

	fmt.Fprintf(cmd.OutOrStdout(), "Switched to context %q.\n", name)
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
		emitJSON(cmd, string(data))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), ctx.Name)
	return nil
}

func runContextCreate(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	}

	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
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

	if createURIFlag != "" {
		rawURI := strings.TrimSpace(createURIFlag)
		if strings.HasPrefix(rawURI, "http://") {
			rawURI = "redis://" + strings.TrimPrefix(rawURI, "http://")
		} else if strings.HasPrefix(rawURI, "https://") {
			rawURI = "rediss://" + strings.TrimPrefix(rawURI, "https://")
		}

		opts, err := redis.ParseURL(rawURI)
		if err != nil {
			return fmt.Errorf("invalid Redis URI: %w", err)
		}

		host, portStr, err := net.SplitHostPort(opts.Addr)
		if err != nil {
			host = opts.Addr
			portStr = "6379"
		}
		port, _ := strconv.Atoi(portStr)
		if port == 0 {
			port = 6379
		}

		newCtx.Host = host
		newCtx.Port = port
		newCtx.Username = opts.Username
		newCtx.Password = opts.Password
		newCtx.DB = opts.DB
		newCtx.TLS = opts.TLSConfig != nil

		if cmd.Flags().Changed("host") {
			newCtx.Host = createHostFlag
		}
		if cmd.Flags().Changed("port") {
			newCtx.Port = createPortFlag
		}
		if cmd.Flags().Changed("db") {
			newCtx.DB = createDBFlag
		}
		if cmd.Flags().Changed("username") {
			newCtx.Username = createUsernameFlag
		}
		if cmd.Flags().Changed("password") {
			newCtx.Password = createPasswordFlag
		}
		if cmd.Flags().Changed("tls") {
			newCtx.TLS = createTLSFlag
		}
	}

	if name == "" {
		// Auto-derive context name if omitted
		if newCtx.Host == "127.0.0.1" || newCtx.Host == "localhost" || newCtx.Host == "::1" {
			name = fmt.Sprintf("local-%d", newCtx.Port)
		} else {
			name = strings.Split(newCtx.Host, ".")[0]
		}
		if name == "" {
			return fmt.Errorf("context name is required. Usage: redis context create <name> [flags]")
		}
	}
	newCtx.Name = name

	if _, exists := cfg.GetContext(name); exists && !forceFlag {
		return fmt.Errorf("context %q already exists (use --force to overwrite)", name)
	}

	newCtx.Supplier = config.DetectSupplier(newCtx.Host, createSupplierFlag)

	cfg.SetContext(newCtx)
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = name
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Context %q created.\n", name)
	return nil
}

func runContextRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load contexts: %w", err)
	}

	targetCtx, exists := cfg.GetContext(oldName)
	if !exists {
		return fmt.Errorf("context %q not found", oldName)
	}

	if oldName != newName {
		if _, destExists := cfg.GetContext(newName); destExists && !forceFlag {
			return fmt.Errorf("context %q already exists (use --force to overwrite)", newName)
		}
		if _, destExists := cfg.GetContext(newName); destExists {
			cfg.DeleteContext(newName)
		}
	}

	ctxCopy := *targetCtx
	ctxCopy.Name = newName
	if renameSupplierFlag != "" {
		ctxCopy.Supplier = renameSupplierFlag
	} else if ctxCopy.Supplier == "" {
		ctxCopy.Supplier = config.DetectSupplier(ctxCopy.Host, "")
	}

	wasCurrent := (cfg.CurrentContext == oldName)
	cfg.DeleteContext(oldName)
	cfg.SetContext(ctxCopy)
	if wasCurrent {
		cfg.CurrentContext = newName
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Renamed context %q to %q.\n", oldName, newName)
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

	fmt.Fprintf(cmd.OutOrStdout(), "Context %q deleted.\n", name)
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

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully imported %d context(s).\n", len(importedCfg.Contexts))
	return nil
}
