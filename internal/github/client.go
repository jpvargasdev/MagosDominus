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
	repo    string 
}

// Commit stages and commits local changes; returns false if nothing to commit.
func (c *Client) Commit(localPath, message string) (bool, error) {
	cmd := exec.Command("git", "-C", localPath, "add", ".")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("git add: %w", err)
	}

	commit := exec.Command("git", "-C", localPath, "commit", "-m", message)
	commit.Stdout, commit.Stderr = os.Stdout, os.Stderr
	if err := commit.Run(); err != nil {
		// exit 1 when there is nothing to commit
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			log.Printf("[github] nothing to commit")
			return false, nil
		}
		return false, fmt.Errorf("git commit: %w", err)
	}

	return true, nil
}

// PushToMain pushes the current HEAD to origin/main using a fresh installation token.
func (c *Client) PushToMain(localPath string) error {
	_, authURL, err := c.authURLs()
	if err != nil {
		return err
	}
	push := exec.Command("git", "-C", localPath, "push", authURL, "HEAD:refs/heads/main")
	push.Stdout, push.Stderr = os.Stdout, os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("git push main failed: %w", err)
	}
	log.Printf("[github] pushed to main")
	return nil
}

// PushAsPR ensures a branch off base, pushes it, and opens a PR; returns PR URL.
func (c *Client) PushAsPR(ctx context.Context, localPath, base, branch, title, body string) (string, error) {
	_, authURL, err := c.authURLs()
	if err != nil {
		return "", err
	}

	// Make sure base is up-to-date locally
	fetch := exec.Command("git", "-C", localPath, "fetch", authURL, base)
	fetch.Stdout, fetch.Stderr = os.Stdout, os.Stderr
	if err := fetch.Run(); err != nil {
		return "", fmt.Errorf("git fetch %s: %w", base, err)
	}

	// Create or reset branch from origin/base
	if err := c.ensureBranchFrom(localPath, base, branch); err != nil {
		return "", err
	}

	// Push branch
	push := exec.Command("git", "-C", localPath, "push", authURL, fmt.Sprintf("HEAD:refs/heads/%s", branch), branch)
	push.Stdout, push.Stderr = os.Stdout, os.Stderr
	if err := push.Run(); err != nil {
		return "", fmt.Errorf("git push branch failed: %w", err)
	}
	log.Printf("[github] pushed branch %s", branch)

	// Open PR
	pr, _, err := c.api.PullRequests.Create(ctx, c.owner(), c.repoName(), &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(branch),
		Base:  github.String(base),
		Body:  github.String(body),
	})
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}
	if pr.HTMLURL == nil {
		return "", fmt.Errorf("create PR: missing URL")
	}
	log.Printf("[github] opened PR: %s", *pr.HTMLURL)
	return *pr.HTMLURL, nil
}

// authURLs returns clean and tokenized remotes.
func (c *Client) authURLs() (cleanURL, authURL string, err error) {
	token, terr := c.itr.Token(context.Background())
	if terr != nil {
		return "", "", fmt.Errorf("github: get token: %w", terr)
	}
	clean := fmt.Sprintf("https://github.com/%s.git", c.repo)
	auth := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, c.repo)
	return clean, auth, nil
}

// ensureBranchFrom creates or resets a local branch from origin/<base>.
func (c *Client) ensureBranchFrom(localPath, base, branch string) error {
	// checkout base
	coBase := exec.Command("git", "-C", localPath, "checkout", "-q", "-B", base, "origin/"+base)
	coBase.Stdout, coBase.Stderr = os.Stdout, os.Stderr
	if err := coBase.Run(); err != nil {
		return fmt.Errorf("checkout base %s: %w", base, err)
	}

	// create/reset branch from base
	co := exec.Command("git", "-C", localPath, "checkout", "-q", "-B", branch, "origin/"+base)
	co.Stdout, co.Stderr = os.Stdout, os.Stderr
	if err := co.Run(); err != nil {
		return fmt.Errorf("create/reset branch %s: %w", branch, err)
	}
	return nil
}

// Small accessors to avoid splitting c.repo again elsewhere
func (c *Client) owner() string {
	parts := strings.SplitN(c.repo, ":", 2) // never hits, safety
	_ = parts
	p := strings.SplitN(c.repo, "/", 2)
	return p[0]
}
func (c *Client) repoName() string {
	p := strings.SplitN(c.repo, "/", 2)
	if len(p) == 2 {
		return p[1]
	}
	return c.repo
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
