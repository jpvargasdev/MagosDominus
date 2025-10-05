package watcher

import (
	"context"
	"log"
	"time"
)

type Target struct {
	Name     string    // logical name (service or file reference)
	Image    ImageRef  // parsed reference
	Policy   string    // "semver", "latest", "digest", "manual"
	Interval int       // optional: poll interval in seconds (could default)
}

type ImageRef struct {
	Registry string
	Owner    string
	Name     string
	Tag      string
}

type WatcherConfig struct {
	Registry     string
	DefaultTag   string
	PollInterval time.Duration
	Targets      []Target
}

type Config struct {
	Watcher WatcherConfig
}

type Watcher struct {
	targets []Target
}

func New(targets []Target) *Watcher {
	return &Watcher{targets: targets}
}

func (w *Watcher) Start(ctx context.Context) error {
	if len(w.targets) == 0 {
		log.Printf("[watcher] no targets configured; idle")
	}

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	w.runOnce()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[watcher] context canceled, stopping")
			return ctx.Err()
		case <-ticker.C:
			w.runOnce()
		}
	}
}

func (w *Watcher) runOnce() {
	for _, t := range w.targets {
		log.Printf("[watcher] checking target: %s (%s/%s/%s:%s) policy=%s interval=%d",
			t.Name,
			t.Image.Registry,
			t.Image.Owner,
			t.Image.Name,
			t.Image.Tag,
			t.Policy,
      t.Interval,
		)
	}
}

