package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	// Wire these when you add real logic:
	// "github.com/youruser/mini-flux/internal/config"
	// "github.com/youruser/mini-flux/internal/watcher"
	// "github.com/youruser/mini-flux/internal/policy"
	// "github.com/youruser/mini-flux/internal/reconciler"
)

var (
	cfgPath   string
	dryRun    bool
	appFilter string
	timeout   time.Duration
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "mini-flux",
	Short: "A tiny GitOps agent for homelabs",
	Long:  "mini-flux watches registries, evaluates image policies, updates your GitOps repo and (optionally) applies changes.",
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/config.yaml", "Path to config file")

	// run (daemon)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	rootCmd.AddCommand(runCmd)

	// reconcile (one-shot)
	reconcileCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	reconcileCmd.Flags().StringVar(&appFilter, "app", "", "Reconcile a single app by name (default: all)")
	reconcileCmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Timeout for one-shot reconcile")
	rootCmd.AddCommand(reconcileCmd)

	// version
	rootCmd.AddCommand(versionCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the mini-flux daemon (watch + reconcile loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// cfg, err := config.Load(cfgPath)
		// if err != nil { return err }

		// ctx with cancellation on SIGINT/SIGTERM
		_, cancel := signalContext()
		defer cancel()

		// Instantiate components (watcher, policy, reconciler) here.
		// st := watcher.NewInMemoryState()
		// factory := func(ep watcher.Endpoint) (watcher.Backend, error) { ... }
		// w := watcher.New(cfg.WatcherConfig, st, factory)
		// events := make(chan watcher.ImageDiscovered, 64)

		// go func() { _ = w.Start(ctx, events) }()

		// for {
		// 	select {
		// 	case <-ctx.Done():
		// 		return ctx.Err()
		// 	case ev := <-events:
		// 		// Evaluate policy → reconcile → optional apply
		// 		// policy.Evaluate(...)
		// 		// reconciler.Reconcile(..., dryRun)
		// 	}
		// }

		fmt.Println("[mini-flux] run: not implemented yet (wire watcher/policy/reconciler)")
		return nil
	},
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Run a one-shot reconcile (no watcher)",
	Long:  "Loads config, evaluates policies against current registry state and patches the GitOps repo once. Useful for CI or manual bumps.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// cfg, err := config.Load(cfgPath)
		// if err != nil { return err }

		_, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Option A (cheap): inspect only the images for the requested app(s).
		// Option B (richer): list tags and choose latest that matches policy.

		// pseudo-code:
		// for _, app := range cfg.Apps {
		//   if appFilter != "" && app.Name != appFilter { continue }
		//   img := app.Image
		//   // resolve candidate (by semver policy)
		//   cand, err := registry.ResolveCandidate(ctx, img, app.Policy, cfg.RegistryAuth)
		//   if err != nil { log.Printf("skip %s: %v", app.Name, err); continue }
		//   approved, reason := policy.Evaluate(ctx, cand, app.Policy)
		//   if !approved {
		//     log.Printf("[policy] %s rejected: %s", app.Name, reason)
		//     continue
		//   }
		//   changed, err := reconciler.PatchAndCommit(ctx, cfg.Git, app, cand.Digest, dryRun)
		//   if err != nil { return err }
		//     if changed && !dryRun && app.ApplyImmediate {
		//       if err := applier.Apply(ctx, cfg.Runtime, app); err != nil { return err }
		//     }
		// }

		log.Printf("[mini-flux] reconcile: app=%q dryRun=%v config=%s (stub)", appFilter, dryRun, cfgPath)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		// Usually injected at build time: -ldflags "-X main.version=..."
		fmt.Println("mini-flux version 0.0.1-dev")
	},
}

// signalContext returns a context canceled on SIGINT/SIGTERM.
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
