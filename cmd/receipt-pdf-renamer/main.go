package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/cache"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
	"github.com/naotama2002/receipt-pdf-renamer/internal/renamer"
	"github.com/naotama2002/receipt-pdf-renamer/internal/tui"
)

var (
	cfgFile    string
	execMode   bool
	pathFlag   string
	dryRun     bool
	clearCache bool
	noCache    bool
	workers    int
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "receipt-pdf-renamer [directory]",
	Short: "Rename receipt PDF files using AI",
	Long: `receipt-pdf-renamer scans PDF files in a directory and renames them
based on the payment date and service name extracted by AI.

Example:
  receipt-pdf-renamer                    # TUI mode, current directory
  receipt-pdf-renamer /path/to/receipts  # TUI mode, specified directory
  receipt-pdf-renamer --exec --path=/path/to/receipts  # Headless mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.Flags().BoolVar(&execMode, "exec", false, "run in headless mode (no UI)")
	rootCmd.Flags().StringVar(&pathFlag, "path", "", "target directory (for --exec mode)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without renaming")
	rootCmd.Flags().BoolVar(&clearCache, "clear-cache", false, "clear the analysis cache")
	rootCmd.Flags().BoolVar(&noCache, "no-cache", false, "disable cache for this run")
	rootCmd.Flags().IntVar(&workers, "workers", 0, "number of parallel workers (overrides config)")
}

func run(cmd *cobra.Command, args []string) error {
	// ディレクトリを先に決定（ローカル設定ファイルの読み込みに必要）
	directory := "."
	if len(args) > 0 {
		directory = args[0]
	}
	if pathFlag != "" {
		directory = pathFlag
	}

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", directory)
	}

	// グローバル設定 + ローカル設定を読み込む
	cfg, err := config.LoadWithLocal(cfgFile, directory)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if workers > 0 {
		cfg.AI.MaxWorkers = workers
	}

	if noCache {
		cfg.Cache.Enabled = false
	}

	cacheInstance, err := cache.New(&cfg.Cache)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	if clearCache {
		fmt.Print("Clear all cached analysis results? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" {
			if err := cacheInstance.Clear(); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}
			fmt.Println("Cache cleared.")
		}
		return nil
	}

	provider, err := ai.NewProvider(&cfg.AI)
	if err != nil {
		return fmt.Errorf("failed to initialize AI provider: %w", err)
	}

	renamerInstance, err := renamer.New(&cfg.Format)
	if err != nil {
		return fmt.Errorf("failed to initialize renamer: %w", err)
	}

	if execMode {
		fmt.Printf("Using %s\n", cfg.ProviderDisplayName())
		fmt.Printf("Model: %s\n\n", cfg.AI.Model)
		return runHeadless(directory, provider, cacheInstance, renamerInstance, cfg.AI.MaxWorkers)
	}

	return runTUI(directory, provider, cacheInstance, renamerInstance, cfg.AI.MaxWorkers, cfg)
}

func runTUI(directory string, provider ai.Provider, cacheInstance *cache.Cache, renamerInstance *renamer.Renamer, maxWorkers int, cfg *config.Config) error {
	configInfo := tui.ConfigInfo{
		ProviderName:   cfg.ProviderDisplayName(),
		Model:          cfg.AI.Model,
		MaxWorkers:     cfg.AI.MaxWorkers,
		CacheEnabled:   cfg.Cache.Enabled,
		ServicePattern: cfg.Format.ServicePattern,
	}
	model := tui.NewModel(directory, provider, cacheInstance, renamerInstance, maxWorkers, configInfo)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func runHeadless(directory string, provider ai.Provider, cacheInstance *cache.Cache, renamerInstance *renamer.Renamer, maxWorkers int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted. Stopping...")
		cancel()
	}()

	runner := tui.NewHeadlessRunner(directory, provider, cacheInstance, renamerInstance, maxWorkers, dryRun)
	result, err := runner.Run(ctx)
	if err != nil {
		return err
	}

	if result.Failed > 0 {
		if result.Renamed > 0 {
			os.Exit(1)
		}
		os.Exit(2)
	}

	return nil
}
