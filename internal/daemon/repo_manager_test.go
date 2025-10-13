package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jpvargasdev/MagosDominus/internal/watcher"
)

func writeFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func normPath(p string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(p, `\`, `/`)
	}
	return p
}

func TestSplitImageRef(t *testing.T) {
	tests := []struct {
		in           string
		wantRegistry string
		wantOwner    string
		wantName     string
		wantTag      string
	}{
		{"ghcr.io/jpvargasdev/lexcodex:0.0.3", "ghcr.io", "jpvargasdev", "lexcodex", "0.0.3"},
		{"ghcr.io/owner/app:latest", "ghcr.io", "owner", "app", "latest"},
		{"ghcr.io/owner/app", "ghcr.io", "owner", "app", "latest"},
		{"badformat", "", "", "", ""},
		{"ghcr.io/only/two", "ghcr.io", "only", "two", "latest"},
	}

	for _, tc := range tests {
		r, o, n, tag := splitImageRef(tc.in)
		if r != tc.wantRegistry || o != tc.wantOwner || n != tc.wantName || tag != tc.wantTag {
			t.Fatalf("splitImageRef(%q) = (%q,%q,%q,%q), want (%q,%q,%q,%q)",
				tc.in, r, o, n, tag, tc.wantRegistry, tc.wantOwner, tc.wantName, tc.wantTag)
		}
	}
}

func TestParseMagosAnnotations_FindsAnnotatedImage(t *testing.T) {
	tmp := t.TempDir()

	yml := `
services:
  lexcodex:
    image: ghcr.io/jpvargasdev/lexcodex:0.0.3 # {"magos":{"policy":"semver","repo":"ghcr.io/jpvargasdev/lexcodex"}}
    container_name: lexcodex
`
	yml2 := `
services:
  other:
    image: ghcr.io/some/other:1.2.3
`
	_ = writeFile(t, tmp, "stacks/lexcodex/lexcodex-compose.yml", strings.TrimLeft(yml, "\n"))
	_ = writeFile(t, tmp, "stacks/other/compose.yml", strings.TrimLeft(yml2, "\n"))
	_ = writeFile(t, tmp, "README.md", "# not yaml")

	rm := &RepoManager{Path: tmp}
	annos, err := rm.ParseMagosAnnotations()
	if err != nil {
		t.Fatalf("ParseMagosAnnotations error: %v", err)
	}
	if len(annos) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annos))
	}
	a := annos[0]
	if !strings.HasSuffix(normPath(a.File), "stacks/lexcodex/lexcodex-compose.yml") {
		t.Fatalf("wrong file: %s", a.File)
	}
	if got, want := a.Image, "ghcr.io/jpvargasdev/lexcodex:0.0.3"; got != want {
		t.Fatalf("image mismatch: got %q want %q", got, want)
	}
	if a.Policy != "semver" {
		t.Fatalf("policy mismatch: got %q want %q", a.Policy, "semver")
	}
	if a.Line <= 0 {
		t.Fatalf("line should be > 0, got %d", a.Line)
	}
}

func TestParseMagosAnnotations_DefaultsManualWhenMissing(t *testing.T) {
	tmp := t.TempDir()

	yml := `
services:
  svc:
    image: ghcr.io/owner/app:1.0.0 # {"magos":{}}
`
	_ = writeFile(t, tmp, "s/compose.yml", strings.TrimLeft(yml, "\n"))

	rm := &RepoManager{Path: tmp}
	annos, err := rm.ParseMagosAnnotations()
	if err != nil {
		t.Fatalf("ParseMagosAnnotations error: %v", err)
	}
	if len(annos) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annos))
	}
	if annos[0].Policy != "manual" {
		t.Fatalf("expected default policy manual, got %q", annos[0].Policy)
	}
}

func TestBuildTargets_MapsFieldsAndSkipsManual(t *testing.T) {
	annos := []MagosAnnotation{
		{
			File:   "/tmp/git/stacks/lexcodex/lexcodex-compose.yml",
			Image:  "ghcr.io/jpvargasdev/lexcodex:0.0.3",
			Policy: "semver",
		},
		{
			File:   "/tmp/git/stacks/skip/compose.yml",
			Image:  "ghcr.io/owner/skip:1.2.3",
			Policy: "manual", // should be skipped
		},
	}

	rm := &RepoManager{Path: "/tmp/git"}
	targets := rm.BuildTargets(annos)

	if len(targets) != 1 {
		t.Fatalf("expected 1 target (manual skipped), got %d", len(targets))
	}

	t0 := targets[0]
	if got, want := t0.Name, "/tmp/git/stacks/lexcodex/lexcodex-compose.yml"; got != want {
		t.Fatalf("Name mismatch: got %q want %q", got, want)
	}
	if t0.Policy != "semver" {
		t.Fatalf("Policy mismatch: %q", t0.Policy)
	}
	if t0.Interval != 0 {
		t.Fatalf("Interval should be 0, got %d", t0.Interval)
	}

	wantImg := watcher.ImageRef{
		Registry: "ghcr.io",
		Owner:    "jpvargasdev",
		Name:     "lexcodex",
		Tag:      "0.0.3",
	}
	if t0.Image != wantImg {
		t.Fatalf("ImageRef mismatch:\n got: %#v\nwant: %#v", t0.Image, wantImg)
	}
}
