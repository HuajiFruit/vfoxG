//go:build !windows

package app

import (
	"os"
	"path/filepath"
)

func (a *App) ensureJunction(linkPath string, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			existing, readErr := os.Readlink(linkPath)
			if readErr == nil && existing == target {
				return nil
			}
		} else if fi.IsDir() {
			if linkPath == target {
				return nil
			}
		}
	}
	_ = os.RemoveAll(linkPath)
	_ = os.MkdirAll(filepath.Dir(linkPath), 0755)

	return os.Symlink(target, linkPath)
}
