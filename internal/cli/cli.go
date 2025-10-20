package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jpvargasdev/magos-dominus/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	// These get overridden via -ldflags at build time.
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	cfgPath string
	dryRun  bool
	verbose bool
	timeout time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "magos-dominus",
	Short: "A tiny GitOps agent for homelabs",
	Long:  "magos-dominus watches registries, evaluates image policies, updates your GitOps repo and (optionally) applies changes.",
}

func Execute() error { return rootCmd.Execute() }

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-ch; cancel() }()
	return ctx, cancel
}

func init() {
	// Version wiring (enables --version)
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(
		"Magos Dominus {{.Version}}\ncommit: " + commit + "\ndate:   " + date + "\n",
	)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/config.yaml", "Path to config file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	// Subcommands
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	rootCmd.AddCommand(runCmd, completionCmd, versionCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the magos-dominus daemon (watch + reconcile loop)",
	PreRun: func(cmd *cobra.Command, _ []string) {
		flags := log.LstdFlags | log.Lmicroseconds
		if verbose {
			flags |= log.Lshortfile
		}
		log.SetFlags(flags)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signalContext()
		defer cancel()
		d := daemon.New(64)
		return d.Start(ctx)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Magos Dominus %s\ncommit: %s\ndate:   %s\n", version, commit, date)
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}
