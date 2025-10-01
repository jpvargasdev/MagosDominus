package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GHCRBackend talks to ghcr.io using the Docker Registry v2 API.
// It supports anonymous access for public images and optional bearer tokens
// via GitHub App installation token or a classic PAT for higher rate limits / private images.
//
// Authentication strategy:
// 1) Try cached bearer for the repo scope.
// 2) If 401 with WWW-Authenticate challenge, call ghcr.io/token without creds (public)
//    or with Basic x-access-token:{installationToken} if provided.
// 3) Retry original request with bearer.

type GHCRBackend struct {
	client            *http.Client
	installationToken string // GitHub App installation token (optional)
	patToken          string // fallback classic PAT (optional)
	tokens            map[string]string // scope -> bearer
}

func NewGHCR(installationToken string) *GHCRBackend {
	return &GHCRBackend{
		client: &http.Client{Timeout: 15 * time.Second},
		installationToken: installationToken,
		patToken: "",
		tokens: make(map[string]string),
	}
}

func (g *GHCRBackend) WithPAT(pat string) *GHCRBackend { g.patToken = pat; return g }

// HeadDigest resolves a reference (tag or digest) to its immutable manifest digest.
func (g *GHCRBackend) HeadDigest(ctx context.Context, repo, reference, platform string) (digest string, tagResolved string, etag string, notModified bool, err error) {
	if strings.HasPrefix(reference, "sha256:") {
		return reference, reference, "", false, nil
	}
	u := fmt.Sprintf("https://ghcr.io/v2/%s/manifests/%s", repo, url.PathEscape(reference))
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
	}, ", "))

	resp, err := g.doAuthed(req, repo, "pull")
	if err != nil { return "", "", "", false, err }
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return "", reference, resp.Header.Get("ETag"), true, nil
	case http.StatusOK:
		if d := resp.Header.Get("Docker-Content-Digest"); d != "" {
			return d, reference, resp.Header.Get("ETag"), false, nil
		}
		// Some registries omit digest on HEAD; do a GET as fallback
		return g.getDigest(ctx, repo, reference)
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return "", "", "", false, fmt.Errorf("ghcr head %s:%s %d: %s", repo, reference, resp.StatusCode, string(b))
	}
}

func (g *GHCRBackend) getDigest(ctx context.Context, repo, reference string) (string, string, string, bool, error) {
	u := fmt.Sprintf("https://ghcr.io/v2/%s/manifests/%s", repo, url.PathEscape(reference))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
	}, ", "))
	resp, err := g.doAuthed(req, repo, "pull")
	if err != nil { return "", "", "", false, err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return "", "", "", false, fmt.Errorf("ghcr get %s:%s %d: %s", repo, reference, resp.StatusCode, string(b))
	}
	if d := resp.Header.Get("Docker-Content-Digest"); d != "" {
		return d, reference, resp.Header.Get("ETag"), false, nil
	}
	return "", "", "", false, fmt.Errorf("digest missing from response headers")
}

// ListTags returns up to pageSize tags; nextToken is currently empty because GHCR's Link-based paging
// is inconsistent and most repos are small; you can extend this later.
func (g *GHCRBackend) ListTags(ctx context.Context, repo string, pageSize int, pageToken string) (tags []string, next string, err error) {
	u := fmt.Sprintf("https://ghcr.io/v2/%s/tags/list?n=%d", repo, pageSize)
	if pageToken != "" { u += "&last=" + url.QueryEscape(pageToken) }
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := g.doAuthed(req, repo, "pull")
	if err != nil { return nil, "", err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return nil, "", fmt.Errorf("ghcr tags %s %d: %s", repo, resp.StatusCode, string(b))
	}
	var out struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, "", err }
	return out.Tags, "", nil
}

func (g *GHCRBackend) doAuthed(req *http.Request, repo, scope string) (*http.Response, error) {
	key := repo + ":" + scope
	if tok := g.tokens[key]; tok != "" {
		r2 := req.Clone(req.Context())
		r2.Header.Set("Authorization", "Bearer "+tok)
		if et := req.Header.Get("If-None-Match"); et != "" { r2.Header.Set("If-None-Match", et) }
		resp, err := g.client.Do(r2)
		if err == nil && resp.StatusCode != http.StatusUnauthorized { return resp, nil }
		if resp != nil { resp.Body.Close() }
	}

	// initial request to get challenge
	resp, err := g.client.Do(req)
	if err != nil { return nil, err }
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	www := resp.Header.Get("Www-Authenticate")
	resp.Body.Close()

	// parse minimal bits from the challenge; GHCR uses service=ghcr.io and scope=repository:<repo>:<scope>
	// we ignore realm since token URL is fixed for ghcr
	tokenURL := fmt.Sprintf("https://ghcr.io/token?service=ghcr.io&scope=repository:%s:%s", repo, scope)
	reqTok, _ := http.NewRequest(http.MethodGet, tokenURL, nil)
	if g.installationToken != "" {
		reqTok.SetBasicAuth("x-access-token", g.installationToken)
	} else if g.patToken != "" {
		reqTok.SetBasicAuth("", g.patToken)
	}
	rTok, err := g.client.Do(reqTok)
	if err != nil { return nil, err }
	defer rTok.Body.Close()
	if rTok.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(rTok.Body, 2<<10))
		return nil, fmt.Errorf("token exchange failed: %d: %s (challenge=%s)", rTok.StatusCode, string(b), www)
	}
	var tok struct{ Token string `json:"token"` }
	if err := json.NewDecoder(rTok.Body).Decode(&tok); err != nil { return nil, err }
	g.tokens[key] = tok.Token

	r2 := req.Clone(req.Context())
	r2.Header.Set("Authorization", "Bearer "+tok.Token)
	return g.client.Do(r2)
}

