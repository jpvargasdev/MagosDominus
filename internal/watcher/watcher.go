package watcher

import (
	"context"
	"log"
	"time"
  "os"
)

type TargetConfig struct {
	Name string 
	Repo string 
	Tag  string 
}

type WatcherConfig struct {
	Registry     string
	DefaultTag   string
	PollInterval time.Duration
	Targets      []TargetConfig
}

type Config struct {
	Watcher WatcherConfig
}

type Watcher struct {
	targets []TargetConfig
}

func New(targets []TargetConfig) *Watcher {
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
		log.Printf("[watcher] checking target: %s (%s:%s)", t.Name, t.Repo, t.Tag)
	}
}

func LoadConfigFromEnv() (*Config, error) {
	cfg := &Config{
		Watcher: WatcherConfig{
			Registry:     os.Getenv("MD_REGISTRY"),
			DefaultTag:   os.Getenv("MD_DEFAULT_TAG"),
			PollInterval: 6 * time.Hour,
			Targets: []TargetConfig{
				{
					Name: "example",
					Repo: "juan/administratus",
					Tag:  "latest",
				},
			},
		},
	}

	return cfg, nil
}
