package watcher

import (
	"context"
	"fmt"
	"log"
	"magos-dominus/internal/state"
  "magos-dominus/internal/events"
	"strings"
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
  emitter events.Emitter 
}

func New(targets []Target, em events.Emitter) *Watcher {
  return &Watcher{targets: targets, emitter: em}
}

func (w *Watcher) Start(ctx context.Context, st *state.File) error {
  ghcr := NewGHCR()
  
	if len(w.targets) == 0 {
		log.Printf("[watcher] no targets configured; idle")
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	w.runOnce(ctx, ghcr, st)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[watcher] context canceled, stopping")
			return ctx.Err()
		case <-ticker.C:
			w.runOnce(ctx, ghcr, st)
		}
	}
}

func (w *Watcher) runOnce(ctx context.Context, ghcr *GHCR, st *state.File) {
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

		repo := fmt.Sprintf("%s/%s", t.Image.Owner, t.Image.Name)
		ref := strings.ToLower(t.Image.Tag)

		key := state.Key(
			strings.ToLower(t.Image.Registry),
			strings.ToLower(t.Image.Owner),
			strings.ToLower(t.Image.Name),
			ref,
		)

		prev, ok := st.Get(key)
		etagIn := ""
		if ok {
			etagIn = prev.ETag
		}

		digest, resolvedRef, etagOut, notMod, err := ghcr.HeadDigest(ctx, repo, ref, etagIn, t.Policy)
    log.Printf("[watcher] repo=%s ref=%s digest=%s", repo, ref, etagOut)
		if err != nil {
			log.Printf("[watcher] skip %s:%s: %v", repo, ref, err)
			continue
		}

		if notMod {
			st.UpdateChecked(key, t.Policy)
			continue
		}

		// no baseline → seed and move on
		if !ok {
			st.UpsertDigest(key, digest, etagOut, t.Policy)
			_ = st.Save()
			log.Printf("[watcher] seeded baseline for %s:%s -> %s", repo, ref, digest)
			continue
		}

		// same digest → just refresh
		if prev.Digest == digest {
			st.UpsertDigest(key, digest, etagOut, t.Policy)
			continue
		}

		// real update
		changed := st.UpsertDigest(key, digest, etagOut, t.Policy)
		if changed {
			log.Printf("[watcher] update: %s:%s -> digest=%s", repo, resolvedRef, digest)
			w.emitter.Emit(events.Event{
				Discovered: time.Now().UTC(),
				File:       t.Name,
				Repo:       repo,
				Ref:        resolvedRef,
				Digest:     digest,
				Policy:     t.Policy,
			})
		} else {
      log.Printf("[watcher] no change: %s:%s -> digest=%s", repo, resolvedRef, digest)
    }
	}
}
