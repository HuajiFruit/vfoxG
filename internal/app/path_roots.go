package app

import (
	"os"
	"path/filepath"
	"strings"
)

func (a *App) vfoxManagedPathRoots() []string {
	var roots []string
	addRoot := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		root = filepath.Clean(root)
		for _, existing := range roots {
			if samePath(existing, root) {
				return
			}
		}
		roots = append(roots, root)
	}

	if vfoxHome := strings.TrimSpace(a.getVfoxHome()); vfoxHome != "" {
		addRoot(filepath.Join(vfoxHome, "cache"))
		addRoot(filepath.Join(vfoxHome, "sdks"))
		addRoot(filepath.Join(vfoxHome, "path-shims"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		legacyHome := filepath.Join(home, ".vfox")
		addRoot(filepath.Join(legacyHome, "cache"))
		addRoot(filepath.Join(legacyHome, "sdks"))
		addRoot(filepath.Join(legacyHome, "path-shims"))
	}
	return roots
}

func (a *App) isVfoxManagedPath(path string) bool {
	for _, root := range a.vfoxManagedPathRoots() {
		if isPathWithin(path, root) {
			return true
		}
	}
	return false
}
