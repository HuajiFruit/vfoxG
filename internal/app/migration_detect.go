package app

import (
	"os"
	"path/filepath"
	"strings"
)

func hasMigratableVfoxHomeData(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		info, err := os.Lstat(entryPath)
		if err != nil {
			return false, err
		}
		if info.IsDir() && !hasDirectoryEntries(entryPath) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func hasDirectoryEntries(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true
	}
	return len(entries) > 0
}
