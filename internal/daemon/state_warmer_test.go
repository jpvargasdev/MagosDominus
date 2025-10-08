package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"magos-dominus/internal/state"
	"magos-dominus/internal/watcher"
)

type fakeBackend struct {
	byRepo map[string]struct {
		digest string
		etag   string
		ref    string
		err    error
	}
}

func (f fakeBackend) HeadDigest(ctx context.Context, repo, ref, etag string) (string, string, string, bool, error) {
	entry, ok := f.byRepo[repo+":"+ref]
	if !ok {
		return "", ref, "", false, os.ErrNotExist
	}
	return entry.digest, entry.ref, entry.etag, false, entry.err
}

func TestWarmState_SeedsEntries(t *testing.T) {
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

	fb := fakeBackend{
		byRepo: map[string]struct {
			digest string
			etag   string
			ref    string
			err    error
		}{
			"jpvargasdev/lexcodex:0.0.3": {digest: "sha256:aaa", etag: `"E1"`, ref: "0.0.3"},
			"owner/app:1.2.3":            {digest: "sha256:bbb", etag: `"E2"`, ref: "1.2.3"},
		},
	}

	if err := warmStateWithBackend(context.Background(), st, targets, fb); err != nil {
		t.Fatalf("warmState error: %v", err)
	}

	key1 := state.Key("ghcr.io", "jpvargasdev", "lexcodex", "0.0.3")
	e1, ok := st.Get(key1)
	if !ok || e1.Digest != "sha256:aaa" || e1.ETag != `"E1"` || e1.Policy != "semver" {
		t.Fatalf("bad entry for key1: ok=%v entry=%+v", ok, e1)
	}

	key2 := state.Key("ghcr.io", "owner", "app", "1.2.3")
	e2, ok := st.Get(key2)
	if !ok || e2.Digest != "sha256:bbb" || e2.ETag != `"E2"` || e2.Policy != "latest" {
		t.Fatalf("bad entry for key2: ok=%v entry=%+v", ok, e2)
	}

	if _, err := os.Stat(filepath.Join(tmp, "state.json")); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
}

func TestWarmState_SkipsOnErrorButSavesOthers(t *testing.T) {
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

	fb := fakeBackend{
		byRepo: map[string]struct {
			digest string
			etag   string
			ref    string
			err    error
		}{
			"ok/good:1.0.0":   {digest: "sha256:okok", etag: `"E-ok"`, ref: "1.0.0"},
			"fail/bad:9.9.9":  {err: os.ErrNotExist},
		},
	}

	if err := warmStateWithBackend(context.Background(), st, targets, fb); err != nil {
		t.Fatalf("warmState error: %v", err)
	}

	keyOK := state.Key("ghcr.io", "ok", "good", "1.0.0")
	if e, ok := st.Get(keyOK); !ok || e.Digest != "sha256:okok" {
		t.Fatalf("expected seeded ok entry, got ok=%v entry=%+v", ok, e)
	}

	keyFail := state.Key("ghcr.io", "fail", "bad", "9.9.9")
	if _, ok := st.Get(keyFail); ok {
		t.Fatalf("did not expect entry for failing target")
	}
}
