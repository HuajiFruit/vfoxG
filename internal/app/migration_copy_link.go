package app

import (
	"os"
	"path/filepath"
)

func (a *App) copyLinkNoOverwriteWithProgress(src string, dst string, oldRoot string, newRoot string, tracker *migrationTracker) error {
	target, err := migrationLinkTarget(src, oldRoot, newRoot)
	if err != nil {
		return err
	}
	if err := a.ensureJunction(dst, target); err != nil {
		return err
	}
	if tracker != nil {
		tracker.finish(src)
	}
	return nil
}

func migrationLinkTarget(src string, oldRoot string, newRoot string) (string, error) {
	target, err := os.Readlink(src)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(src), target)
	}
	target = filepath.Clean(target)
	if !isPathWithin(target, oldRoot) {
		return target, nil
	}

	rel, err := filepath.Rel(oldRoot, target)
	if err != nil {
		return "", err
	}
	return filepath.Join(newRoot, rel), nil
}
