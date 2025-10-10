package github 

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v75/github"
)

type Client struct {
	api     *github.Client
	itr     *ghinstallation.Transport
	repo    string // always "owner/repo" after normalization
}

func New(appID, installationID int64, privateKeyPath, repo string) *Client {
	r, err := normalizeRepo(repo)
	if err != nil {
		log.Fatalf("github: bad repo %q: %v", repo, err)
	}
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, installationID, privateKeyPath)
	if err != nil {
		log.Fatalf("github: installation transport error: %v", err)
	}
	api := github.NewClient(&http.Client{Transport: itr})
	return &Client{api: api, itr: itr, repo: r}
}

// normalizeRepo converts SSH/HTTPS forms into "owner/repo".
func normalizeRepo(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".git")
	switch {
	case strings.HasPrefix(s, "git@github.com:"):
		s = strings.TrimPrefix(s, "git@github.com:")
	case strings.HasPrefix(s, "https://github.com/"):
		s = strings.TrimPrefix(s, "https://github.com/")
	}
	if !strings.Contains(s, "/") {
		return "", fmt.Errorf("expected owner/repo, got %q", s)
	}
	return s, nil
}

// CloneOrPull uses a fresh installation token each call.
func (c *Client) CloneOrPull(localPath string) error {
	token, err := c.itr.Token(context.Background())
	if err != nil {
		return fmt.Errorf("github: get token: %w", err)
	}
	cleanURL := fmt.Sprintf("https://github.com/%s.git", c.repo)
	authURL  := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, c.repo)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		log.Printf("[github] cloning %s into %s", cleanURL, localPath)
		cmd := exec.Command("git", "clone", authURL, localPath)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone failed: %w", err)
		}
		// sanitize to avoid storing tokens in .git/config
		return exec.Command("git", "-C", localPath, "remote", "set-url", "origin", cleanURL).Run()
	}

	log.Printf("[github] pulling latest changes in %s", localPath)
	// for private repos use authURL; for public it'll also work
	fetch := exec.Command("git", "-C", localPath, "fetch", authURL, "main")
	fetch.Stdout, fetch.Stderr = os.Stdout, os.Stderr
	if err := fetch.Run(); err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}
	co := exec.Command("git", "-C", localPath, "checkout", "main")
	co.Stdout, co.Stderr = os.Stdout, os.Stderr
	if err := co.Run(); err != nil {
		return fmt.Errorf("checkout main failed: %w", err)
	}
	pull := exec.Command("git", "-C", localPath, "pull", "--ff-only", authURL, "main")
	pull.Stdout, pull.Stderr = os.Stdout, os.Stderr
	return pull.Run()
}
