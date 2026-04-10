package vault

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolvePath resolves relPath inside vaultPath and rejects traversal outside the vault root.
func ResolvePath(vaultPath, relPath string) (string, error) {
	rootAbs, err := filepath.Abs(vaultPath)
	if err != nil {
		return "", fmt.Errorf("resolving vault path: %w", err)
	}

	full := filepath.Clean(filepath.Join(rootAbs, relPath))
	rel, err := filepath.Rel(rootAbs, full)
	if err != nil {
		return "", fmt.Errorf("validating path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes vault root")
	}
	return full, nil
}
