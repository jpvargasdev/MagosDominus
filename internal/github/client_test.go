package github

import (
	"os/exec"
	"testing"
)

// --- normalizeRepo -----------------------------------------------------------

func TestNormalizeRepo_HTTPS(t *testing.T) {
	got, err := normalizeRepo("https://github.com/owner/repo")
	if err != nil {
		t.Fatalf("normalizeRepo error: %v", err)
	}
	if got != "owner/repo" {
		t.Fatalf("want owner/repo, got %q", got)
	}
}

func TestNormalizeRepo_SSH(t *testing.T) {
	got, err := normalizeRepo("git@github.com:owner/repo.git")
	if err != nil {
		t.Fatalf("normalizeRepo error: %v", err)
	}
	if got != "owner/repo" {
		t.Fatalf("want owner/repo, got %q", got)
	}
}

func TestNormalizeRepo_Bad(t *testing.T) {
	if _, err := normalizeRepo("no-slash-here"); err == nil {
		t.Fatalf("expected error for bad repo string")
	}
}

// --- helpers ----------------------------------------------------------------

func must(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd %v failed: %v\n%s", cmd.Args, err, string(out))
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	must(t, exec.Command("git", "-C", dir, "init", "-q"))
	must(t, exec.Command("git", "-C", dir, "config", "user.name", "Test Bot"))
	must(t, exec.Command("git", "-C", dir, "config", "user.email", "test-bot@example.com"))
	// quiet the trustworthy whining in some CI
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", dir).Run()
}

