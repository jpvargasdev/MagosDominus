package daemon

import (
  "fmt"
  "os"
  "strings"
)
// UpdateImage rewrites the image reference in a compose file.
// If useDigest is true, it pins as "<repo>@<digest>"; otherwise "<repo>:<ref>".
// It only writes if the content actually changes.
// Returns updated=true if the file was modified.
func (r *RepoManager) UpdateImage(filePath, newRef, newDigest string, useDigest bool) (bool, error) {
	// 1) read file
	src, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(src), "\n")

	// 2) scan for the image line that also carries the magos annotation
	//    (keeps things conservative; won’t touch unrelated images)
	updated := false
	for i, line := range lines {
		if !strings.Contains(line, "image:") || !strings.Contains(line, `{"magos"`) {
			continue
		}

		// left = before "#", right = after "#"
		left, right, ok := strings.Cut(line, "#")
		if !ok {
			continue
		}

		// extract current image ref from "image: <ref>"
		img := strings.TrimSpace(left)
		if idx := strings.Index(img, "image:"); idx >= 0 {
			img = strings.TrimSpace(img[idx+len("image:"):])
		}
		if img == "" {
			continue
		}

		// base repo (strip tag or digest)
		base := stripRefOrDigest(img)

		// build desired ref
		var desired string
		if useDigest {
			// require a digest-looking value
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

		// idempotency: if already desired, skip write
		if normalizeImage(img) == normalizeImage(desired) {
			continue
		}

		// recompose line, preserving the annotation tail
		newLine := fmt.Sprintf("    image: %s #%s", desired, right)
		lines[i] = newLine
		updated = true
		// optional: break after first match; remove this if multiple services in same file
		break
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
