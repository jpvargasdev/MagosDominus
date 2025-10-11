package github

import (
	"os"
	"os/exec"
	"path/filepath"
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

// --- Commit -----------------------------------------------------------------

func TestCommit_CommitsWhenDirty_AndNoopsWhenClean(t *testing.T) {
	// local client object; Commit doesn't use api/itr/repo
	c := &Client{}

	tmp := t.TempDir()
	initGitRepo(t, tmp)

	// Create an initial file
	fp := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(fp, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// First commit should succeed and return changed=true
	changed, err := c.Commit(tmp, "add a.txt")
	if err != nil {
		t.Fatalf("Commit error: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true on first commit")
	}

	// Second commit without changes should return changed=false, no error
	changed, err = c.Commit(tmp, "noop")
	if err != nil {
		t.Fatalf("Commit error on noop: %v", err)
	}
	if changed {
		t.Fatalf("expected changed=false on noop")
	}
}

// --- ensureBranchFrom with local bare remote --------------------------------

func TestEnsureBranchFrom_CreatesBranchFromBase(t *testing.T) {
	// Arrange a bare remote and a working clone so "origin/base" exists.
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	must(t, exec.Command("git", "init", "--bare", remoteDir))

	// Seed a temp working repo, make an initial commit on main, push to bare.
	seedDir := filepath.Join(t.TempDir(), "seed")
	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, seedDir)

	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	must(t, exec.Command("git", "-C", seedDir, "add", "."))
	must(t, exec.Command("git", "-C", seedDir, "commit", "-m", "seed"))

	// Set remote and push main
	must(t, exec.Command("git", "-C", seedDir, "branch", "-M", "main"))
	must(t, exec.Command("git", "-C", seedDir, "remote", "add", "origin", remoteDir))
	must(t, exec.Command("git", "-C", seedDir, "push", "-u", "origin", "main"))

	// Fresh working clone that will run ensureBranchFrom
	workDir := filepath.Join(t.TempDir(), "work")
	must(t, exec.Command("git", "clone", remoteDir, workDir))

	// Client with only repo string set (api/itr unused by ensureBranchFrom)
	c := &Client{repo: "owner/repo"}

	// Act: ensure a feature branch from base "main"
	if err := c.ensureBranchFrom(workDir, "main", "magos/test-branch"); err != nil {
		t.Fatalf("ensureBranchFrom error: %v", err)
	}

	// Assert current branch is magos/test-branch and based on origin/main
	out, err := exec.Command("git", "-C", workDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if string(bytesTrim(out)) != "magos/test-branch" {
		t.Fatalf("expected to be on magos/test-branch, got %q", string(out))
	}

	// Verify the branch points to same commit as origin/main initially
	head, _ := exec.Command("git", "-C", workDir, "rev-parse", "HEAD").Output()
	base, _ := exec.Command("git", "-C", workDir, "rev-parse", "origin/main").Output()
	if string(bytesTrim(head)) != string(bytesTrim(base)) {
		t.Fatalf("branch is not based on origin/main")
	}
}

func bytesTrim(b []byte) []byte {
	// trim trailing newline without pulling another import
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
