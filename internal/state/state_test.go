package state

import (
	"path/filepath"
	"testing"
	"time"
)

func tmpFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "state.json")
}

func TestLoadAndSave(t *testing.T) {
	path := tmpFile(t)
	s := New(path)
	s.UpsertDigest("ghcr.io/foo/bar:latest", "sha256:a", "etag1", "manual")
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := New(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	e, ok := s2.Get("ghcr.io/foo/bar:latest")
	if !ok {
		t.Fatalf("missing entry after load")
	}
	if e.Digest != "sha256:a" {
		t.Errorf("digest mismatch: got %q", e.Digest)
	}
}

func TestUpsertDigestChange(t *testing.T) {
	s := New(tmpFile(t))
	key := "ghcr.io/foo/bar:latest"
	changed := s.UpsertDigest(key, "sha256:a", "etag1", "semver")
	if !changed {
		t.Errorf("expected first insert to be change=true")
	}
	changed = s.UpsertDigest(key, "sha256:a", "etag1", "semver")
	if changed {
		t.Errorf("expected no change for same digest")
	}
	changed = s.UpsertDigest(key, "sha256:b", "etag2", "semver")
	if !changed {
		t.Errorf("expected change for new digest")
	}
}

func TestUpdateChecked(t *testing.T) {
	s := New(tmpFile(t))
	key := "ghcr.io/foo/bar:latest"
	s.UpsertDigest(key, "sha256:a", "", "manual")
	before, _ := s.Get(key)
	time.Sleep(10 * time.Millisecond)
	s.UpdateChecked(key, "")
	after, _ := s.Get(key)
	if !after.LastChecked.After(before.LastChecked) {
		t.Errorf("expected LastChecked to advance")
	}
}
