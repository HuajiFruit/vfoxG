package app

import (
	"fmt"
	"os"
)

func (a *App) copyPathNoOverwrite(src string, dst string, oldRoot string, newRoot string) error {
	return a.copyPathNoOverwriteWithProgress(src, dst, oldRoot, newRoot, nil)
}

func (a *App) copyPathNoOverwriteWithProgress(src string, dst string, oldRoot string, newRoot string, tracker *migrationTracker) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	} else if !os.IsNotExist(err) {
		return err
	}

	if tracker != nil {
		tracker.start(src)
	}

	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 {
		return a.copyLinkNoOverwriteWithProgress(src, dst, oldRoot, newRoot, tracker)
	}
	if info.IsDir() {
		return a.copyDirNoOverwriteWithProgress(src, dst, info.Mode().Perm(), oldRoot, newRoot, tracker)
	}
	if err := copyFileNoOverwrite(src, dst, info.Mode().Perm()); err != nil {
		return err
	}
	if tracker != nil {
		tracker.finish(src)
	}
	return nil
}
