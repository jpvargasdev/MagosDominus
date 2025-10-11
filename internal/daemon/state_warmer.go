package daemon

import (
	"context"
	"log"
	"strings"

	"magos-dominus/internal/state"
	"magos-dominus/internal/watcher"
)

var newBackend = func() *watcher.GHCR {
	return watcher.NewGHCR() // kept around if you still use it elsewhere
}

func warmState(st *state.File, targets []watcher.Target) error {
	// no backend; pure local seeding
	for _, t := range targets {
		ref := strings.ToLower(t.Image.Tag)
		key := state.Key(
			strings.ToLower(t.Image.Registry),
			strings.ToLower(t.Image.Owner),
			strings.ToLower(t.Image.Name),
			ref,
		)

		// seed placeholder: empty digest/etag, record policy
		st.UpsertDigest(key, "", "", t.Policy)
	}
	if err := st.Save(); err != nil {
		log.Printf("[warm] state save failed: %v", err)
		return err
	}
	return nil
}

// If you still want a separate helper, keep this no-op wrapper.
func warmStateWithBackend(
	_ context.Context,
	st *state.File,
	targets []watcher.Target,
	_ *watcher.GHCR, // intentionally unused
) error {
	for _, t := range targets {
		ref := strings.ToLower(t.Image.Tag)
		key := state.Key(
			strings.ToLower(t.Image.Registry),
			strings.ToLower(t.Image.Owner),
			strings.ToLower(t.Image.Name),
			ref,
		)
		st.UpsertDigest(key, "", "", t.Policy)
	}
	if err := st.Save(); err != nil {
		log.Printf("[warm] state save failed: %v", err)
		return err
	}
	return nil
}
