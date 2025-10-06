package daemon

import (
	"context"
	"log"
	"magos-dominus/internal/events"
	"magos-dominus/internal/state"
	"magos-dominus/internal/watcher"
)

type Daemon struct {
  events events.ChanEmitter
}

func New(buffer int) *Daemon {
	return &Daemon{
    events: make(events.ChanEmitter, buffer),
  }
}

func (d *Daemon) EventsEmitter() events.Emitter {
	return d.events
}

func (d *Daemon) consume(ctx context.Context) {
  for {
    select {
    case <-ctx.Done():
      return
    case ev := <-d.events:
      log.Printf("[event] repo=%s ref=%s digest=%s", ev.Repo, ev.Ref, ev.Digest)
      // 1. rm.Sync()
      // 2. rm.UpdateImage(ev.File, ev.Digest)
      // 3. rm.CommitAndPush()
      // 4. Run reconcile.sh
    }
  }
}

func (d *Daemon) Start(ctx context.Context) error {
	log.Printf("[daemon] starting...")
  path := "tmp/magos/state.json"

  // 0. Init Magos state
  st := state.New(path)

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
	w := watcher.New(targets, d.EventsEmitter())

  go d.consume(ctx)

	return w.Start(ctx, st)
}
