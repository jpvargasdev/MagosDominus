package daemon

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type RepoManager struct {
	URL  string
	Path string
}

func NewRepoManager() *RepoManager {
	cfg := getRepo()
  log.Printf("[repo] URL: %s", cfg.RepoURL)

	repoPath := filepath.Join(os.TempDir(), "git")

	return &RepoManager{
		URL:  cfg.RepoURL,
		Path: repoPath,
	}
}

func (r *RepoManager) Sync() error {
	if _, err := os.Stat(r.Path); os.IsNotExist(err) {
		log.Printf("[repo] cloning %s into %s", r.URL, r.Path)
		cmd := exec.Command("git", "clone", r.URL, r.Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	log.Printf("[repo] pulling latest changes in %s", r.Path)
	cmd := exec.Command("git", "-C", r.Path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
