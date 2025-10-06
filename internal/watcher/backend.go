package watcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
  "encoding/json"
)

type GHCR struct {
  client *http.Client
  mu      sync.Mutex
  tokens  map[string]string
}

func NewGHCR() *GHCR {
  return &GHCR{
    client: http.DefaultClient,
    tokens: make(map[string]string),
  }
}

func (g *GHCR) HeadDigest(ctx context.Context, repo, ref, etag string) (string, string, string, bool, error) {
  repo = strings.ToLower(repo)

  token, err := g.tokenFor(ctx, repo)
  if err != nil {
    return "", "", "", false, fmt.Errorf("token: %w", err)
  }
  
  url := fmt.Sprintf("https://ghcr.io/v2/%s/manifests/%s", repo, ref)

  req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
  if err != nil {
    return "", "", "", false, fmt.Errorf("new request: %w", err)
  }

  req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
  req.Header.Set("Accept",
    strings.Join([]string {
      "application/vnd.docker.distribution.manifest.list.v2+json",
			"application/vnd.docker.distribution.manifest.v2+json",
			"application/vnd.oci.image.index.v1+json",
			"application/vnd.oci.image.manifest.v1+json",
    }, ", "),
  )

  if etag != "" {
    req.Header.Set("If-None-Match", etag)
  }

  resp, err := g.client.Do(req)
  if err != nil {
    return "", "", "", false, fmt.Errorf("request: %w", err)
  }

  defer resp.Body.Close()

  switch resp.StatusCode {
    case http.StatusNotModified: // 304
		  return "", ref, etag, true, nil
    case http.StatusOK: // 200
      digest := resp.Header.Get("Docker-Content-Digest")
      etagOut := resp.Header.Get("Etag")
      if digest == "" && etagOut == "" {
        return "", "", "", false, fmt.Errorf("no digest/etag in response headers")
      }
      return digest, ref, etagOut, false, nil
    case http.StatusUnauthorized: // token likely stale; drop and retry once
      g.dropToken(repo)
      return "", "", "", false, fmt.Errorf("unauthorized; token expired/invalid")
    default:
      return "", "", "", false, fmt.Errorf("unexpected status %d", resp.StatusCode)
    }
}

func (g *GHCR) tokenFor(ctx context.Context, repo string) (string, error) {
	g.mu.Lock()
	if tok, ok := g.tokens[repo]; ok && tok != "" {
		g.mu.Unlock()
		return tok, nil
	}
	g.mu.Unlock()

	// Anonymous pull token
	url := fmt.Sprintf("https://ghcr.io/token?scope=repository:%s:pull", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Token == "" {
		return "", fmt.Errorf("empty token from GHCR")
	}

	g.mu.Lock()
	g.tokens[repo] = payload.Token
	g.mu.Unlock()
	return payload.Token, nil
}

func (g *GHCR) dropToken(repo string) {
	g.mu.Lock()
	delete(g.tokens, repo)
	g.mu.Unlock()
}
