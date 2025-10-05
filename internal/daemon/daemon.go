package daemon

import (
	"context"
	"log"
	"magos-dominus/internal/watcher"
)

type Daemon struct {
	cfg *watcher.Config
}

func New(cfg *watcher.Config) *Daemon {
	return &Daemon{cfg: cfg}
}

func (d *Daemon) Start(ctx context.Context) error {
	log.Printf("[daemon] starting...")

  //  1. Sync GitOps repo 
  rm := NewRepoManager()
  if err := rm.Sync(); err != nil {
    return err
  }
  log.Printf("[repo] synced at %s", rm.Path)


  // 2. Create and start watcher with current targets
	w := watcher.New(d.cfg.Watcher.Targets)

	return w.Start(ctx)
}
