package daemon

import (
	"context"
	"log"
	"magos-dominus/internal/watcher"
)

type Daemon struct {
}

func New() *Daemon {
	return &Daemon{}
}

func (d *Daemon) Start(ctx context.Context) error {
	log.Printf("[daemon] starting...")

  //  1. Sync GitOps repo 
  rm := NewRepoManager()
  if err := rm.Sync(); err != nil {
    return err
  }
  log.Printf("[repo] synced at %s", rm.Path)


  // 2. Parse magos annotations
  annotations, err := rm.ParseMagosAnnotations()
  if err != nil {
    return err
  }

  targets := rm.BuildTargets(annotations)
  log.Printf("[repo] find %d targets", len(targets))
  
  // 3. Create and start watcher with current targets
	w := watcher.New(targets)

	return w.Start(ctx)
}
