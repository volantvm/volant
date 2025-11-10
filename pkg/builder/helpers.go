// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"os"
	"path/filepath"
	"strings"
)

// ensureStateDir gets or creates the BuildKit state directory
func ensureStateDir() string {
	// Check environment variable first
	if v := strings.TrimSpace(os.Getenv("VOLANT_BUILDKIT_STATE_DIR")); v != "" {
		abs, err := filepath.Abs(v)
		if err == nil {
			if err := os.MkdirAll(abs, 0o700); err == nil {
				return abs
			}
		}
	}

	// Try user cache directory
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		path := filepath.Join(cacheDir, "volant", "buildkit")
		if err := os.MkdirAll(path, 0o700); err == nil {
			return path
		}
	}

	// Fallback to temp directory
	path := filepath.Join(os.TempDir(), "volant-buildkit")
	os.MkdirAll(path, 0o700)
	return path
}
