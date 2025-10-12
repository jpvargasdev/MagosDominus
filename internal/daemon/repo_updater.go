package daemon

import (
  "fmt"
  "os"
  "strings"
)

func (r *RepoManager) UpdateImage(filePath, newRef, newDigest string, policy string) (bool, error) {
	// 1) read file
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(src), "\n")

	// 2) scan for the image line that also carries the magos annotation
	updated := false
	for i, line := range lines {
		if !strings.Contains(line, "image:") || !strings.Contains(line, `{"magos"`) {
			continue
		}

		left, right, ok := strings.Cut(line, "#")
		if !ok {
			continue
		}

		// extract current image ref from "image: <ref>"
		imgField := strings.TrimRight(left, " \t")
		idx := strings.Index(imgField, "image:")
		if idx < 0 {
			continue
		}
		prefix := imgField[:idx]                 // keep original indentation/prefix
		rest := strings.TrimSpace(imgField[idx:]) // starts with "image:"
		cur := strings.TrimSpace(strings.TrimPrefix(rest, "image:"))
		if cur == "" {
			continue
		}

		// base repo (strip tag or digest)
		base := stripRefOrDigest(cur)

		// build desired ref
		var desired string
		if policy == "digest" {
			if !strings.HasPrefix(newDigest, "sha256:") {
				return false, fmt.Errorf("invalid digest %q", newDigest)
			}
			desired = fmt.Sprintf("%s@%s", base, newDigest)
		} else {
			if newRef == "" {
				return false, fmt.Errorf("empty ref")
			}
			desired = fmt.Sprintf("%s:%s", base, newRef)
		}

		// idempotency: already desired
		if normalizeImage(cur) == normalizeImage(desired) {
			continue
		}

		// recompose line, preserving prefix and annotation tail
		newLine := fmt.Sprintf("%simage: %s #%s", prefix, desired, right)
		lines[i] = newLine
		updated = true
		break // remove if you want to update multiple services in one file
	}

	if !updated {
		return false, nil
	}

	// 3) write back atomically
	tmp := filePath + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, filePath); err != nil {
		return false, err
	}
	return true, nil
}


// stripRefOrDigest returns "registry/owner/name" from an image like
// "ghcr.io/owner/name:tag" or "ghcr.io/owner/name@sha256:...".
func stripRefOrDigest(img string) string {
	img = strings.TrimSpace(img)
	if at := strings.IndexByte(img, '@'); at >= 0 {
		return img[:at]
	}
	// beware digests contain ":"; only strip the *last* colon as tag delimiter
	if c := strings.LastIndexByte(img, ':'); c > 0 && !strings.Contains(img[c+1:], "/") {
		return img[:c]
	}
	return img
}

// normalizeImage helps equality by lowercasing repo part, leaving digest/tag intact.
func normalizeImage(img string) string {
	img = strings.TrimSpace(img)
	if img == "" {
		return img
	}
	// split into repo + ref/digest
	if at := strings.IndexByte(img, '@'); at >= 0 {
		return strings.ToLower(img[:at]) + img[at:]
	}
	if c := strings.LastIndexByte(img, ':'); c > 0 && !strings.Contains(img[c+1:], "/") {
		return strings.ToLower(img[:c]) + img[c:]
	}
	return strings.ToLower(img)
}
