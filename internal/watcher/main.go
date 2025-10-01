package watcher

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

  "magos-dominus/internal/watcher"
)

// main is the thin composition root. It wires env config, backend, state, and starts the watcher.
// Policy/reconciler/applier will be added here later, consuming events from the emitter channel.
func main() {
	// Basic logger setup
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Load env-based config
	cfg, err := watcher.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Cancellation on SIGINT/SIGTERM
	ctx, cancel := signalContext()
	defer cancel()

	// Backend: GHCR using a pre-provisioned installation token for now.
	// Later you’ll hot-reload this via a token refresher built from the GitHub App private key.
	instTok := os.Getenv("GITHUB_APP_INSTALLATION_TOKEN")
	if instTok == "" {
		if cfg.GitHubApp.AppID > 0 {
			log.Printf("[warn] GITHUB_APP_INSTALLATION_TOKEN missing; private repos will fail until token refresher is implemented")
		}
	}
	backend := watcher.NewGHCR(instTok)

	// State & emitter
	state := watcher.NewInMemoryState()
	events := make(watcher.ChanEmitter, 128)

	// Translate config targets into watcher targets
	ttargets := make([]watcher.Target, 0, len(cfg.Watcher.Targets))
	for _, t := range cfg.Watcher.Targets {
		// registry is currently global; extend to per-target if you insist on suffering
		repo := t.Repo // owner/repo only
		if cfg.Watcher.Registry != "" {
			repo = repo // GHCR backend expects repo without registry; keep as-is
		}
		ttargets = append(ttargets, watcher.Target{
			Name: t.Name,
			Image: watcher.ImageRef{
				Registry: cfg.Watcher.Registry,
				Owner:    ownerPart(repo),
				Name:     namePart(repo),
			},
			Interval: t.Interval,
			MinAge:   0,
		})
	}

	w := watcher.New(backend, state, events, ttargets)

	go func() {
		for ev := range events {
			log.Printf("[event] target=%s repo=%s tag=%s digest=%s", ev.Target, ev.Repository, ev.Tag, ev.Digest)
			// TODO: call policy + reconciler here
		}
	}()

	if err := w.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("watcher stopped: %v", err)
	}
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
		// give in-flight requests a grace period before hard exit
		time.Sleep(200 * time.Millisecond)
	}()
	return ctx, cancel
}

func ownerPart(repo string) string {
	// repo is owner/name
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return repo[:i]
		}
	}
	return ""
}

func namePart(repo string) string {
	for i := len(repo) - 1; i >= 0; i-- {
		if repo[i] == '/' {
			return repo[i+1:]
		}
	}
	return repo
}

