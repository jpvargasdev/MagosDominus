package watcher

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

func New(b Backend, s State, e Emitter, targets []Target) *Watcher {
	return &Watcher{backend: b, state: s, out: e, trs: targets}
}

func (w *Watcher) Start(ctx context.Context) error {
	if len(w.trs) == 0 {
		log.Printf("[watcher] no targets configured; idle")
	}
	w.wg.Add(len(w.trs))
	for _, t := range w.trs {
		go func(t Target) {
			defer w.wg.Done()
			w.runTarget(ctx, t)
		}(t)
	}
	<-ctx.Done()
	w.wg.Wait()
	return ctx.Err()
}

func (w *Watcher) runTarget(ctx context.Context, t Target) {
	base := t.Interval
	if base <= 0 {
		base = 30 * time.Second
	}
	// initial jitter to avoid stampedes
	jitter := time.Duration(rand.Int63n(int64(base / 3)))
	timer := time.NewTimer(jitter)
	defer timer.Stop()

	backoff := time.Second // starts tiny, capped below
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if ok := w.tickOnce(ctx, t); ok {
				// success path: reset backoff and schedule next normal tick with tiny jitter
				backoff = time.Second
				next := base + time.Duration(rand.Int63n(int64(base/5)+1))
				timer.Reset(next)
			} else {
				// error path: exponential backoff with cap
				backoff = backoff * 2
				if backoff > 2*time.Minute { backoff = 2 * time.Minute }
				next := backoff + time.Duration(rand.Int63n(int64(backoff/3)+1))
				log.Printf("[watcher] %s backing off for %s", t.Name, next)
				timer.Reset(next)
			}
		}
	}
}

// tickOnce performs a single poll of one target. Returns true on success (including 304/not-changed).
func (w *Watcher) tickOnce(ctx context.Context, t Target) bool {
	repo := t.Image.Owner + "/" + t.Image.Name
	ref := "latest"
	// A policy module will eventually choose the tag; for now, watch a stable marker tag.

	digest, tag, _, notMod, err := w.backend.HeadDigest(ctx, repo, ref, "")
	if err != nil {
		log.Printf("[watcher] %s: head failed for %s:%s: %v", t.Name, repo, ref, err)
		return false
	}
	if notMod {
		return true
	}

	last, ok := w.state.LastDigest(repo, tag, "")
	if ok && last == digest {
		return true
	}

	w.state.SetDigest(repo, tag, "", digest)
	w.out.Emit(Event{Target: t.Name, Repository: repo, Tag: tag, Digest: digest, Discovered: time.Now()})
	return true
}

// InMemoryState is a trivial map-based State implementation for bootstrapping.
type InMemoryState struct {
	mu sync.Mutex
	m  map[string]string
}

func NewInMemoryState() *InMemoryState { return &InMemoryState{m: make(map[string]string)} }

func (s *InMemoryState) LastDigest(repo, tag, platform string) (string, bool) {
	s.mu.Lock(); defer s.mu.Unlock()
	d, ok := s.m[repo+"|"+tag+"|"+platform]
	return d, ok
}

func (s *InMemoryState) SetDigest(repo, tag, platform, digest string) {
	s.mu.Lock(); defer s.mu.Unlock()
	s.m[repo+"|"+tag+"|"+platform] = digest
}

// ChanEmitter is a channel-backed emitter.
type ChanEmitter chan Event

func (c ChanEmitter) Emit(e Event) { c <- e }

