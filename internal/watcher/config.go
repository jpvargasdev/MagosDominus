package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime settings for the watcher and related systems.
// Source of truth is environment variables; a future loader can merge YAML.
//
// Env overview (all optional unless marked REQUIRED):
//   GITHUB_APP_ID                    (REQUIRED for GHCR private or GitHub App auth)
//   GITHUB_APP_INSTALLATION_ID       (REQUIRED when using GitHub App)
//   GITHUB_APP_PRIVATE_KEY           (PEM content; mutually exclusive with *_PATH)
//   GITHUB_APP_PRIVATE_KEY_PATH      (path to PEM; mutually exclusive with *_PRIVATE_KEY)
//
//   GIT_REPO                         (e.g. "github.com/juan/repo")
//   GIT_BRANCH                       (default: "main")
//   GIT_SSH_KEY_PATH                 (optional; if pushing via SSH)
//
//   REGISTRY                         (default: "ghcr.io")
//   WATCH_TARGETS                    comma-separated list of targets. Each target:
//                                    "owner/repo[:tagHint][@interval][#platform][|name]"
//                                    Examples:
//                                      "acme/whoami:1.x@30s#linux/amd64|whoami"
//                                      "org/svc:latest@1m"
//                                    tagHint defaults to "latest"; interval to 30s; platform empty
//   WATCH_DEFAULT_INTERVAL           (fallback interval, e.g. "30s")
//
// Notes:
// - If you only use public images on GHCR, GitHub App vars are optional.
// - Git settings are used by the reconciler, but we keep them here so CLI can validate early.

func LoadConfigFromEnv() (Config, error) {
	var cfg Config

	// GitHub App
	appID := getenvInt64("GITHUB_APP_ID", 0)
	instID := getenvInt64("GITHUB_APP_INSTALLATION_ID", 0)
	pem := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	pemPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	if pem != "" && pemPath != "" {
		return cfg, errors.New("set either GITHUB_APP_PRIVATE_KEY or GITHUB_APP_PRIVATE_KEY_PATH, not both")
	}
	if (appID > 0 || instID > 0 || pem != "" || pemPath != "") && (appID == 0 || instID == 0 || (pem == "" && pemPath == "")) {
		return cfg, errors.New("GitHub App auth incomplete: require GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and either PRIVATE_KEY or PRIVATE_KEY_PATH")
	}
	cfg.GitHubApp = GitHubApp{AppID: appID, InstallationID: instID}
	if pem != "" {
		cfg.GitHubApp.PrivateKeyPEM = []byte(pem)
	} else if pemPath != "" {
		cfg.GitHubApp.PrivateKeyPath = pemPath
	}

	// Git
	cfg.Git = GitConfig{
		Repo:       os.Getenv("GIT_REPO"),
		Branch:     getenvDefault("GIT_BRANCH", "main"),
		SSHKeyPath: os.Getenv("GIT_SSH_KEY_PATH"),
	}

	// Watcher
	cfg.Watcher.Registry = getenvDefault("REGISTRY", "ghcr.io")
	cfg.Watcher.DefaultInt = getenvDuration("WATCH_DEFAULT_INTERVAL", 30*time.Second)

	targetsStr := strings.TrimSpace(os.Getenv("WATCH_TARGETS"))
	if targetsStr != "" {
		parsed, err := parseTargets(targetsStr, cfg.Watcher.DefaultInt)
		if err != nil { return cfg, err }
		cfg.Watcher.Targets = parsed
	}

	return cfg, nil
}

func parseTargets(spec string, defInterval time.Duration) ([]Target, error) {
	parts := splitComma(spec)
	out := make([]Target, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" { continue }

		var name string
		if i := strings.Index(p, "|"); i >= 0 {
			name = strings.TrimSpace(p[i+1:])
			p = p[:i]
		}

		platform := ""
		if i := strings.Index(p, "#"); i >= 0 {
			platform = strings.TrimSpace(p[i+1:])
			p = p[:i]
		}

		interval := defInterval
		if i := strings.Index(p, "@"); i >= 0 {
			ivalStr := strings.TrimSpace(p[i+1:])
			p = p[:i]
			if d, err := time.ParseDuration(ivalStr); err == nil { interval = d } else { return nil, fmt.Errorf("invalid interval %q in %q: %w", ivalStr, p, err) }
		}

		repo := p
		tag := "latest"
		if i := strings.Index(p, ":"); i >= 0 {
			tag = strings.TrimSpace(p[i+1:])
			repo = strings.TrimSpace(p[:i])
		}

		if !strings.Contains(repo, "/") {
			return nil, fmt.Errorf("target %q must be owner/repo", repo)
		}
		if name == "" {
			name = filepath.Base(repo)
		}
		out = append(out, Target{
			Name: name, Repo: repo, TagHint: tag, Platform: platform, Interval: interval,
		})
	}
	return out, nil
}

// Helpers
func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" { return def }
	return v
}

func getenvInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" { return def }
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil { return def }
	return i
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" { return def }
	d, err := time.ParseDuration(v)
	if err != nil { return def }
	return d
}

func splitComma(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if t := strings.TrimSpace(r); t != "" {
			out = append(out, t)
		}
	}
	return out
}

