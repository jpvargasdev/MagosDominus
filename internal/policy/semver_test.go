package policy

import (
	"testing"
)

func TestResolveSemver_BasicLatest(t *testing.T) {
	tags := []string{"main", "v0.0.3", "0.0.4"}
	got, err := ResolveSemver(tags)
	if err != nil {
		t.Fatalf("ResolveSemver error: %v", err)
	}
	if got != "0.0.4" {
		t.Fatalf("want 0.0.4, got %q", got)
	}
}

func TestResolveSemver_KeepsOriginalSpelling(t *testing.T) {
	tags := []string{"main", "v1.2.0", "v1.2.3"}
	got, err := ResolveSemver(tags)
	if err != nil {
		t.Fatalf("ResolveSemver error: %v", err)
	}
	if got != "v1.2.3" {
		t.Fatalf("want v1.2.3 (preserve v-prefix), got %q", got)
	}
}

func TestResolveSemver_StableBeatsPrerelease(t *testing.T) {
	tags := []string{"0.1.0-rc.1", "0.1.0-rc.2", "0.1.0"}
	got, err := ResolveSemver(tags)
	if err != nil {
		t.Fatalf("ResolveSemver error: %v", err)
	}
	if got != "0.1.0" {
		t.Fatalf("want 0.1.0 (stable beats prerelease), got %q", got)
	}
}

func TestResolveSemver_OnlyPrereleases(t *testing.T) {
	tags := []string{"v2.0.0-beta.1", "v2.0.0-beta.2"}
	got, err := ResolveSemver(tags)
	if err != nil {
		t.Fatalf("ResolveSemver error: %v", err)
	}
	if got != "v2.0.0-beta.2" {
		t.Fatalf("want v2.0.0-beta.2, got %q", got)
	}
}

func TestResolveSemver_IgnoresNonSemver(t *testing.T) {
	tags := []string{"main", "latest", "develop"}
	_, err := ResolveSemver(tags)
	if err == nil {
		t.Fatalf("expected error when no semver tags are present")
	}
}

func TestResolveSemver_WhitespaceAndEmpty(t *testing.T) {
	tags := []string{"  v3.1.4  ", "", "  "}
	got, err := ResolveSemver(tags)
	if err != nil {
		t.Fatalf("ResolveSemver error: %v", err)
	}
	if got != "v3.1.4" {
		t.Fatalf("want v3.1.4, got %q", got)
	}
}
