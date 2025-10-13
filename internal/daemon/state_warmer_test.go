package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jpvargasdev/Administratus/internal/state"
	"github.com/jpvargasdev/Administratus/internal/watcher"
)

func TestWarmState_SeedsEntriesWithEmptyValues(t *testing.T) {
	tmp := t.TempDir()
	st := state.New(filepath.Join(tmp, "state.json"))

	targets := []watcher.Target{
		{
			Name: "/tmp/git/stacks/lexcodex/compose.yml",
			Image: watcher.ImageRef{
				Registry: "ghcr.io",
				Owner:    "jpvargasdev",
				Name:     "lexcodex",
				Tag:      "0.0.3",
			},
			Policy: "semver",
		},
		{
			Name: "/tmp/git/stacks/other/compose.yml",
			Image: watcher.ImageRef{
				Registry: "ghcr.io",
				Owner:    "owner",
				Name:     "app",
				Tag:      "1.2.3",
			},
			Policy: "latest",
		},
	}

	if err := warmState(st, targets); err != nil {
		t.Fatalf("warmState error: %v", err)
	}

	key1 := state.Key("ghcr.io", "jpvargasdev", "lexcodex", "0.0.3")
	e1, ok := st.Get(key1)
	if !ok {
		t.Fatalf("missing entry for key1")
	}
	if e1.Digest != "" || e1.ETag != "" {
		t.Fatalf("expected empty digest/etag for key1, got digest=%q etag=%q", e1.Digest, e1.ETag)
	}
	if e1.Policy != "semver" {
		t.Fatalf("expected policy=semver for key1, got %q", e1.Policy)
	}

	key2 := state.Key("ghcr.io", "owner", "app", "1.2.3")
	e2, ok := st.Get(key2)
	if !ok {
		t.Fatalf("missing entry for key2")
	}
	if e2.Digest != "" || e2.ETag != "" {
		t.Fatalf("expected empty digest/etag for key2, got digest=%q etag=%q", e2.Digest, e2.ETag)
	}
	if e2.Policy != "latest" {
		t.Fatalf("expected policy=latest for key2, got %q", e2.Policy)
	}

	if _, err := os.Stat(filepath.Join(tmp, "state.json")); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
}

func TestWarmState_SeedsAllTargets_NoSkips(t *testing.T) {
	tmp := t.TempDir()
	st := state.New(filepath.Join(tmp, "state.json"))

	targets := []watcher.Target{
		{
			Name: "/tmp/git/a.yml",
			Image: watcher.ImageRef{
				Registry: "ghcr.io",
				Owner:    "ok",
				Name:     "good",
				Tag:      "1.0.0",
			},
			Policy: "semver",
		},
		{
			Name: "/tmp/git/b.yml",
			Image: watcher.ImageRef{
				Registry: "ghcr.io",
				Owner:    "fail",
				Name:     "bad",
				Tag:      "9.9.9",
			},
			Policy: "semver",
		},
	}

	if err := warmState(st, targets); err != nil {
		t.Fatalf("warmState error: %v", err)
	}

	keyOK := state.Key("ghcr.io", "ok", "good", "1.0.0")
	if e, ok := st.Get(keyOK); !ok || e.Policy == "" {
		t.Fatalf("expected seeded ok entry with policy, got ok=%v entry=%+v", ok, e)
	}

	keyOther := state.Key("ghcr.io", "fail", "bad", "9.9.9")
	if e, ok := st.Get(keyOther); !ok || e.Policy == "" {
		t.Fatalf("expected seeded second entry with policy, got ok=%v entry=%+v", ok, e)
	}
}
