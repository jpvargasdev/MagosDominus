package state

 import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry captures the last-known info for a single image reference.
type Entry struct {
	Digest      string    `json:"digest"`
	ETag        string    `json:"etag,omitempty"`
	Policy      string    `json:"policy,omitempty"`
	LastChecked time.Time `json:"lastChecked"`
	LastChanged time.Time `json:"lastChanged"`
}

// File is a JSON-backed state store.
type File struct {
	path string
	mu   sync.Mutex
	// key: "<registry>/<owner>/<name>:<ref>"
	entries map[string]Entry
}

// New creates a new File state store; Load must be called to populate from disk.
func New(path string) *File {
	return &File{
		path:    path,
		entries: make(map[string]Entry),
	}
}

// Load reads state from disk if present; creates parent dir if missing.
func (f *File) Load() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.path == "" {
		return errors.New("state: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f.entries = make(map[string]Entry)
			return nil
		}
		return err
	}
	var onDisk struct {
		Version int               `json:"version"`
		Entries map[string]Entry  `json:"entries"`
	}
	if err := json.Unmarshal(data, &onDisk); err != nil {
		return err
	}
	if onDisk.Entries == nil {
		onDisk.Entries = make(map[string]Entry)
	}
	f.entries = onDisk.Entries
	return nil
}

// Save writes state to disk atomically.
func (f *File) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.path == "" {
		return errors.New("state: empty path")
	}
	payload := struct {
		Version   int               `json:"version"`
		UpdatedAt time.Time         `json:"updatedAt"`
		Entries   map[string]Entry  `json:"entries"`
	}{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Entries:   f.entries,
	}

	tmp := f.path + ".tmp"
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

// Key builds the canonical key for an image ref.
func Key(registry, owner, name, ref string) string {
	// registry and repo names in GHCR are case-insensitive; normalize to lower.
	return filepath.ToSlash((registry + "/" + owner + "/" + name + ":" + ref))
}

// Get returns the entry for a key (registry/owner/name:ref).
func (f *File) Get(key string) (Entry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.entries[key]
	return e, ok
}

// UpdateChecked updates the LastChecked timestamp without changing digest.
func (f *File) UpdateChecked(key string, policy string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e := f.entries[key]
	e.Policy = policyOrKeep(e.Policy, policy)
	e.LastChecked = time.Now().UTC()
	f.entries[key] = e
}

// UpsertDigest sets digest/etag; returns true if digest changed.
func (f *File) UpsertDigest(key, digest, etag, policy string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC()
	e := f.entries[key]

	changed := e.Digest != digest && digest != ""
	if changed {
		e.Digest = digest
		e.LastChanged = now
	}
	if etag != "" {
		e.ETag = etag
	}
	e.Policy = policyOrKeep(e.Policy, policy)
	e.LastChecked = now
	f.entries[key] = e
	return changed
}

func policyOrKeep(current, incoming string) string {
	if incoming != "" {
		return incoming
	}
	return current
}
