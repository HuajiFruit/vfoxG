package app

import (
	"os"
	"path/filepath"
)

func (a *App) copyDirNoOverwrite(src string, dst string, perm os.FileMode, oldRoot string, newRoot string) error {
	return a.copyDirNoOverwriteWithProgress(src, dst, perm, oldRoot, newRoot, nil)
}

func (a *App) copyDirNoOverwriteWithProgress(src string, dst string, perm os.FileMode, oldRoot string, newRoot string, tracker *migrationTracker) error {
	if err := os.MkdirAll(dst, perm); err != nil {
		return err
	}
	if tracker != nil {
		tracker.finish(src)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if err := a.copyPathNoOverwriteWithProgress(srcPath, dstPath, oldRoot, newRoot, tracker); err != nil {
			return err
		}
	}
	return os.Chmod(dst, perm)
}
