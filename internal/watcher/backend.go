package watcher

import (
	"context"
	"log"
)

type Backend interface {
	HeadDigest(ctx context.Context, repo, tag, etag string) (digest, resolvedTag, etagOut string, notModified bool, err error)
}

type EchoBackend struct {
	Version string
}

func NewEchoBackend(version string) *EchoBackend {
	return &EchoBackend{Version: version}
}

func (e *EchoBackend) HeadDigest(ctx context.Context, repo, tag, etag string) (string, string, string, bool, error) {
	log.Printf("[backend] version=%s repo=%s tag=%s", e.Version, repo, tag)
	return "sha256:dummy", tag, "", false, nil
}
