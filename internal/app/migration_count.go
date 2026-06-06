package app

import (
	"os"
	"path/filepath"
)

func countMigrationItems(root string) (int, error) {
	total := 0
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		count, err := countMigrationPathItems(filepath.Join(root, entry.Name()))
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func countMigrationPathItems(path string) (int, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	total := 1
	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 || !info.IsDir() {
		return total, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		count, err := countMigrationPathItems(filepath.Join(path, entry.Name()))
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}
