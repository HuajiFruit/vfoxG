//go:build !windows

package app

import (
	"os"
	"path/filepath"
	"strings"
)

const vfoxPathMarkerLabel = "vfoxG PATH"

func (a *App) checkVfoxInPath() (bool, error) {
	coreDir := filepath.Clean(a.getCoreDir())
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(strings.TrimSpace(entry)) == coreDir {
			return true, nil
		}
	}
	if unixManagedBlockExists(vfoxPathMarkerLabel) {
		return true, nil
	}
	return false, nil
}

func (a *App) addVfoxToPath() error {
	coreDir, err := a.getVfoxExecutable()
	if err != nil {
		return err
	}
	return unixWritePathBlock(vfoxPathMarkerLabel, []string{filepath.Dir(coreDir)})
}

func (a *App) removeVfoxFromPath() error {
	return unixRemoveManagedBlock(vfoxPathMarkerLabel)
}
