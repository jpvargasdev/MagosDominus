package daemon

import (
	"context"
	"log"
	"strings"

	"magos-dominus/internal/state"
	"magos-dominus/internal/watcher"
)

var newBackend = func() interface {
	HeadDigest(ctx context.Context, repo, ref, etag string) (digest, resolvedRef, etagOut string, notModified bool, err error)
} {
	return watcher.NewGHCR()
}

func warmState(ctx context.Context, st *state.File, targets []watcher.Target) error {
	return warmStateWithBackend(ctx, st, targets, newBackend())
}

func warmStateWithBackend(
	ctx context.Context,
	st *state.File,
	targets []watcher.Target,
	b interface {
		HeadDigest(ctx context.Context, repo, ref, etag string) (digest, resolvedRef, etagOut string, notModified bool, err error)
	},
) error {
	for _, t := range targets {
		repo := strings.ToLower(t.Image.Owner + "/" + t.Image.Name)
		ref := strings.ToLower(t.Image.Tag)

		key := state.Key(
			strings.ToLower(t.Image.Registry),
			strings.ToLower(t.Image.Owner),
			strings.ToLower(t.Image.Name),
			ref,
		)

		digest, resolvedRef, etagOut, _, err := b.HeadDigest(ctx, repo, ref, "")
		if err != nil {
			log.Printf("[warm] skip %s:%s: %v", repo, ref, err)
			continue
		}

		st.UpsertDigest(key, digest, etagOut, t.Policy)
		log.Printf("[warm] baseline set %s:%s digest=%s", repo, resolvedRef, digest)
	}

	if err := st.Save(); err != nil {
		log.Printf("[warm] state save failed: %v", err)
		return err
	}
	return nil
}
