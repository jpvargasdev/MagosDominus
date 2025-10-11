package policy

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// semverPattern matches tags like "v1.2.3" or "1.2.3" (optionally with suffixes like -beta)
var semverPattern = regexp.MustCompile(`^v?(\d+\.\d+\.\d+([\-+].*)?)$`)

// ResolveSemver takes a list of tags and returns the latest semantic version.
// It ignores non-semver tags (e.g. "main", "latest") and returns an error if none found.
func ResolveSemver(tags []string) (string, error) {
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags provided")
	}

	var versions []*semver.Version
	tagMap := make(map[string]string)

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		m := semverPattern.FindStringSubmatch(tag)
		if len(m) == 0 {
			continue
		}

		v, err := semver.NewVersion(m[1])
		if err != nil {
			continue
		}

		versions = append(versions, v)
		tagMap[v.Original()] = tag // keep the exact tag (with/without "v")
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semver tags found in list")
	}

	sort.Sort(semver.Collection(versions))
	latest := versions[len(versions)-1]
	return tagMap[latest.Original()], nil
}
