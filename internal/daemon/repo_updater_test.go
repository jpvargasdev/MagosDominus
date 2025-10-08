package daemon 

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	return p
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	return string(b)
}

func TestUpdateImage_PinByDigest(t *testing.T) {
	tmp := t.TempDir()
	orig := `
services:
  lexcodex:
    image: ghcr.io/jpvargasdev/lexcodex:0.0.3 # {"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}
`
	fp := writeTemp(t, tmp, "compose.yml", strings.TrimLeft(orig, "\n"))

	rm := &RepoManager{Path: tmp}
	ref := "0.0.4"
	digest := "sha256:deadbeefcafebabe0123456789abcdef0123456789abcdef0123456789abcd"

	updated, err := rm.UpdateImage(fp, ref, digest, true) // useDigest=true
	if err != nil {
		t.Fatalf("UpdateImage error: %v", err)
	}
	if !updated {
		t.Fatalf("expected updated=true")
	}

	got := readFile(t, fp)
	if !strings.Contains(got, "image: ghcr.io/jpvargasdev/lexcodex@"+digest) {
		t.Fatalf("expected digest pin, got:\n%s", got)
	}
	if !strings.Contains(got, `{"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}`) {
		t.Fatalf("annotation lost:\n%s", got)
	}
}

func TestUpdateImage_UpdateTag(t *testing.T) {
	tmp := t.TempDir()
	orig := `
services:
  lexcodex:
    image: ghcr.io/jpvargasdev/lexcodex:0.0.3 # {"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}
`
	fp := writeTemp(t, tmp, "compose.yml", strings.TrimLeft(orig, "\n"))

	rm := &RepoManager{Path: tmp}
	updated, err := rm.UpdateImage(fp, "0.0.4", "", false) // useDigest=false
	if err != nil {
		t.Fatalf("UpdateImage error: %v", err)
	}
	if !updated {
		t.Fatalf("expected updated=true")
	}

	got := readFile(t, fp)
	if !strings.Contains(got, "image: ghcr.io/jpvargasdev/lexcodex:0.0.4") {
		t.Fatalf("expected tag 0.0.4, got:\n%s", got)
	}
}

func TestUpdateImage_Idempotent_NoChange(t *testing.T) {
	tmp := t.TempDir()
	orig := `
services:
  lexcodex:
    image: ghcr.io/jpvargasdev/lexcodex@sha256:deadbeef # {"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}
`
	fp := writeTemp(t, tmp, "compose.yml", strings.TrimLeft(orig, "\n"))

	rm := &RepoManager{Path: tmp}
	updated, err := rm.UpdateImage(fp, "ignored", "sha256:deadbeef", true)
	if err != nil {
		t.Fatalf("UpdateImage error: %v", err)
	}
	if updated {
		t.Fatalf("expected updated=false when value already matches")
	}
}

func TestUpdateImage_NoAnnotatedImage_NoOp(t *testing.T) {
	tmp := t.TempDir()
	orig := `
services:
  other:
    image: ghcr.io/some/other:1.2.3
`
	fp := writeTemp(t, tmp, "compose.yml", strings.TrimLeft(orig, "\n"))

	rm := &RepoManager{Path: tmp}
	updated, err := rm.UpdateImage(fp, "2.0.0", "sha256:xyz", false)
	if err != nil {
		t.Fatalf("UpdateImage error: %v", err)
	}
	if updated {
		t.Fatalf("expected updated=false when no Magos annotation present")
	}
}

func TestUpdateImage_InvalidDigest(t *testing.T) {
	tmp := t.TempDir()
	orig := `
services:
  lexcodex:
    image: ghcr.io/jpvargasdev/lexcodex:0.0.3 # {"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}
`
	fp := writeTemp(t, tmp, "compose.yml", strings.TrimLeft(orig, "\n"))

	rm := &RepoManager{Path: tmp}
	if _, err := rm.UpdateImage(fp, "", "not-a-digest", true); err == nil {
		t.Fatalf("expected error for invalid digest")
	}
}

