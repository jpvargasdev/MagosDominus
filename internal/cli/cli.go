package internal

import (
	"context"
	"log"
	"magos-dominus/internal/daemon"
	"magos-dominus/internal/watcher"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	cfgPath   string
	dryRun    bool
	appFilter string
	timeout   time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "magos-dominus",
	Short: "A tiny GitOps agent for homelabs",
	Long:  "magos-dominus watches registries, evaluates image policies, updates your GitOps repo and (optionally) applies changes.",
}

func Execute() error {
	return rootCmd.Execute()
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() { 
		<-ch 
		cancel() 
	}()
	return ctx, cancel
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/config.yaml", "Path to config file")

	// run (daemon)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the magos-dominus daemon (watch + reconcile loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		cfg, err := watcher.LoadConfigFromEnv() 
		if err != nil {
			return err
		}

		ctx, cancel := signalContext()
		defer cancel()

		d := daemon.New(cfg) 
		return d.Start(ctx) 
	},
}

