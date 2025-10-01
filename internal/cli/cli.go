package cli

import (
  "log"
  "os"
  "os/signal"
  "fmt"
  "context"
  "time"
  "syscall"

  "github.com/spf13/cobra"
  "magos-dominus/internal/watcher"
)

var (
	cfgPath   string // kept for future YAML overlays; envs are used today
	dryRun    bool
	appFilter string
	timeout   time.Duration
)


var rootCmd = &cobra.Command{
	Use:   "magos-dominus",
	Short: "A tiny GitOps agent for homelabs",
	Long:  "magos-dominus watches registries, evaluates image policies, updates your GitOps repo and (optionally) applies changes.",
}


func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "configs/config.yaml", "Path to config file")

	// run (daemon)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	rootCmd.AddCommand(runCmd)

	// reconcile (one-shot) — stays stubby for now
	reconcileCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not push/apply; log intended actions")
	reconcileCmd.Flags().StringVar(&appFilter, "app", "", "Reconcile a single app by name (default: all)")
	reconcileCmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Timeout for one-shot reconcile")
	rootCmd.AddCommand(reconcileCmd)

	// version
	rootCmd.AddCommand(versionCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the magos-dominus daemon (watch + reconcile loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		// Carga configuración desde variables de entorno
		cfg, err := watcher.LoadConfigFromEnv()
		if err != nil {
			return err
		}

		// Contexto cancelable con señales (Ctrl+C)
		ctx, cancel := signalContext()
		defer cancel()

		// Backend GHCR en modo anónimo (solo funciona con imágenes públicas)
		b := watcher.NewGHCR("")

		// Estado en memoria y canal de eventos
		state := watcher.NewInMemoryState()
		events := make(watcher.ChanEmitter, 128)

		// Construcción de targets a partir del config
		var targets []watcher.Target
		for _, t := range cfg.Watcher.Targets {
			targets = append(targets, watcher.Target{
				Name:     t.Name,
				Image:    watcher.ImageRef{Registry: cfg.Watcher.Registry, Owner: ownerPart(t.Repo), Name: namePart(t.Repo)},
				Interval: t.Interval,
			})
		}

		// Instanciamos el watcher
		w := watcher.New(b, state, events, targets)

		// Consumidor de eventos
		go func() {
			for ev := range events {
				log.Printf("[event] target=%s repo=%s tag=%s digest=%s",
					ev.Target, ev.Repository, ev.Tag, ev.Digest)
			}
		}()

		log.Printf("[startup] registry=%s targets=%d", cfg.Watcher.Registry, len(targets))
		return w.Start(ctx)
	},
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Run a one-shot reconcile (no watcher)",
	Long:  "Carga la config desde variables de entorno, revisa las imágenes públicas y aplica la política en un solo ciclo.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		cfg, err := watcher.LoadConfigFromEnv()
		if err != nil {
			return err
		}

		// Contexto con timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Backend GHCR anónimo (solo imágenes públicas)
		b := watcher.NewGHCR("")

		for _, t := range cfg.Watcher.Targets {
			repo := ownerPart(t.Repo) + "/" + namePart(t.Repo)
			digest, tag, _, _, err := b.HeadDigest(ctx, repo, t.TagHint, "")
			if err != nil {
				log.Printf("[reconcile] %s: error al resolver %s:%s: %v",
					t.Name, repo, t.TagHint, err)
				continue
			}
			log.Printf("[reconcile] target=%s repo=%s tag=%s digest=%s",
				t.Name, repo, tag, digest)
		}

		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("magos-dominus version 0.0.1-dev")
	},
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

// namePart extrae el nombre del repo de un string tipo "owner/repo".
func namePart(repo string) string {
	for i := len(repo) - 1; i >= 0; i-- {
		if repo[i] == '/' {
			return repo[i+1:]
		}
	}
	return repo
}

func ownerPart(repo string) string {
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return repo[:i]
		}
	}
	return ""
}
