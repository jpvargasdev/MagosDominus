package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"magos-dominus/internal/watcher"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RepoManager struct {
	URL  string
	Path string
}

type MagosAnnotation struct {
  File   string
  Line   int
  Image  string
  Policy string
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

func (r *RepoManager) ParseMagosAnnotations() ([]MagosAnnotation, error) {
	var out []MagosAnnotation

	err := filepath.WalkDir(r.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() { return nil }
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" { return nil }

		f, err := os.Open(path)
		if err != nil { return fmt.Errorf("open %s: %w", path, err) }
		defer f.Close()

		sc := bufio.NewScanner(f)
		ln := 0
		for sc.Scan() {
			ln++
			line := sc.Text()
			if !strings.Contains(line, "image:") || !strings.Contains(line, `{"magos":`) {
				continue
			}

			left, right, ok := strings.Cut(line, "#")
			if !ok { continue }

			img := strings.TrimSpace(left)
			if idx := strings.Index(img, "image:"); idx >= 0 {
				img = strings.TrimSpace(img[idx+len("image:"):])
			}
			if img == "" { continue }

			raw := strings.TrimSpace(right)
			jsonStart := strings.Index(raw, "{")
			if jsonStart < 0 { continue }
			raw = raw[jsonStart:]

			var payload struct {
				Magos struct {
					Policy string `json:"policy"`
					Note   string `json:"note"`
				} `json:"magos"`
			}
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				continue
			}
			policy := strings.TrimSpace(payload.Magos.Policy)
			if policy == "" { policy = "manual" } 

			out = append(out, MagosAnnotation{
				File:   path,
				Line:   ln,
				Image:  img,
				Policy: policy,
			})
		}
		return sc.Err()
	})

	return out, err
}

func (r *RepoManager) BuildTargets(annos []MagosAnnotation) []watcher.Target {
	var targets []watcher.Target
	for _, a := range annos {
		if a.Policy == "manual" {
			continue
		}
		registry, owner, name, tag := splitImageRef(a.Image)
		targets = append(targets, watcher.Target{
			Name: a.File, 
			Image: watcher.ImageRef{
        Registry: registry,
        Owner:    owner,
        Name:     name,
        Tag:      tag,
      },
      Policy: a.Policy,
      Interval: 0,
		})
	}
	return targets
}

func splitImageRef(img string) (string, string, string, string) {
	// supports something like ghcr.io/repo/app:0.0.1
	parts := strings.SplitN(img, "/", 3)
	if len(parts) < 3 {
		return "", "", "", ""
	}
	registry, owner, rest := parts[0], parts[1], parts[2]
	nameTag := strings.SplitN(rest, ":", 2)
	name, tag := nameTag[0], "latest"
	if len(nameTag) == 2 {
		tag = nameTag[1]
	}
	return registry, owner, name, tag
}
