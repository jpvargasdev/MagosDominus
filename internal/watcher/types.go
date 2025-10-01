package watcher

import (
  "time"
  "context"
  "sync"
)

type Config struct {
	GitHubApp GitHubApp
	Git       GitConfig
	Watcher   WatcherConfig
}

type GitHubApp struct {
	AppID          int64
	InstallationID int64
	// Exactly one of the following will be set
	PrivateKeyPEM  []byte
	PrivateKeyPath string
}

type GitConfig struct {
	Repo       string // e.g. "github.com/juan/repo" or "git@github.com:juan/repo.git"
	Branch     string // default main
	SSHKeyPath string // optional
}

type WatcherConfig struct {
	Registry   string
	DefaultInt time.Duration
	Targets    []Target
}

// Target mirrors watcher.Target minimally to avoid import cycles.
// Name is optional; if empty, derived from repo name.
type Target struct {
	Name     string
	Repo     string // owner/repo, no registry
	TagHint  string // e.g. "latest" or semver family hint like "1.x" (policy will evolve)
	Platform string // e.g. "linux/amd64" (optional)
	Interval time.Duration
  Image    ImageRef
}

// Backend is implemented by GHCRBackend (and future registries) to resolve digests and list tags.
type Backend interface {
	HeadDigest(ctx context.Context, repo, reference, platform string) (digest string, tagResolved string, etag string, notModified bool, err error)
	ListTags(ctx context.Context, repo string, pageSize int, pageToken string) (tags []string, next string, err error)
}

// State stores the last observed digest per (repo, tag, platform).
type State interface {
	LastDigest(repo, tag, platform string) (string, bool)
	SetDigest(repo, tag, platform, digest string)
}

// Emitter pushes discovered update events to downstream consumers (policy/reconciler).
type Emitter interface { Emit(Event) }

// Event is emitted when a repo:tag resolved to a new digest.
type Event struct {
	Target     string
	Repository string
	Tag        string
	Digest     string
	Discovered time.Time
}

// ImageRef identifies an image (split is mostly for future multi-registry support).
type ImageRef struct {
	Registry string // e.g. "ghcr.io"
	Owner    string // e.g. "juanvargas"
	Name     string // e.g. "myapp"
}

// Watcher orchestrates per-target polling with light backoff and jitter.
type Watcher struct {
	backend Backend
	state   State
	out     Emitter
	trs     []Target

	wg sync.WaitGroup
}
