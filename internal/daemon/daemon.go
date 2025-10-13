package daemon

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jpvargasdev/Administratus/internal/config"
	"github.com/jpvargasdev/Administratus/internal/events"
	"github.com/jpvargasdev/Administratus/internal/reconciler"
	"github.com/jpvargasdev/Administratus/internal/state"
	"github.com/jpvargasdev/Administratus/internal/watcher"
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

func (d *Daemon) consume(ctx context.Context, rm *RepoManager) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-d.events:
			cfg := config.GetGitPreferences()

			log.Printf("[event] repo=%s ref=%s digest=%s", ev.Repo, ev.Ref, ev.Digest)

			// 1) sync
			if err := rm.Sync(); err != nil {
				log.Printf("[error] repo sync: %v", err)
				continue
			}

			// 2) update image in the specific file
			changed, err := rm.UpdateImage(ev.File, ev.Ref, ev.Digest, ev.Policy)
			if err != nil {
				log.Printf("[error] update image: %v", err)
				continue
			}
			if !changed {
				log.Printf("[event] no changes")
				continue
			}

			log.Printf("[event] updated %s", ev.File)

			// 3) commit & push (or PR) — one file per event
			if err := rm.CommitAndPush(ev.File, cfg.PreferPR); err != nil {
				log.Printf("[error] commit and push: %v", err)
				continue
			}

			// 4) reconcile hook (placeholder)
			log.Printf("[event] running reconcile.sh")
			if err := reconciler.RunReconcile(ctx, os.Getenv("MD_RECONCILE_SCRIPT"), rm.Path, ev.File, ev.Policy); err != nil {
				log.Printf("[error] reconcile: %v", err)
			}
		}
	}
}

func (d *Daemon) Start(ctx context.Context) error {
	log.Printf("[daemon] starting...")

	// 0. Init Magos state
	path := "tmp/magos/state.json"
	st := state.New(path)
	if err := st.Load(); err != nil {
		return fmt.Errorf("state load: %w", err)
	}

	//  1. Sync GitOps repo
	rm := NewRepoManager()
	if err := rm.SyncFresh(); err != nil {
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

	// 3. Warm state from repo
	if err := warmState(st, targets); err != nil {
		log.Printf("[warm] failed: %v", err)
	}

	// 4. Initial run
	paths := rm.BuildReconcilePaths(annotations)
	reconciler.RunAll(ctx, os.Getenv("MD_RECONCILE_SCRIPT"), rm.Path, paths)

	// 5. Create and start watcher with current targets
	go d.consume(ctx, rm)
	w := watcher.New(targets, d.EventsEmitter())
	return w.Start(ctx, st)
}
