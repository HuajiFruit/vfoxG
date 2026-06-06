package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) migrateVfoxHomeData(from string, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" || samePath(from, to) {
		return nil
	}
	if isPathWithin(to, from) {
		return fmt.Errorf("new download path cannot be inside the current vfox data directory")
	}
	if err := os.MkdirAll(to, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(from)
	if err != nil {
		return err
	}
	total, err := countMigrationItems(from)
	if err != nil {
		return err
	}
	startedAt := time.Now()
	a.emitMigrationProgress("preparing", "", 0, total, startedAt)
	tracker := &migrationTracker{
		app:       a,
		total:     total,
		startedAt: startedAt,
		oldRoot:   from,
	}
	for _, entry := range entries {
		name := entry.Name()
		src := filepath.Join(from, name)
		if err := a.copyPathNoOverwriteWithProgress(src, filepath.Join(to, name), from, to, tracker); err != nil {
			a.emitMigrationProgress("error", tracker.current, tracker.completed, total, startedAt)
			return fmt.Errorf("failed to migrate %s: %w", name, err)
		}
	}
	a.emitMigrationProgress("done", "", total, total, startedAt)
	a.emitEvent("vfox-log", "[INFO] Migrated vfox SDK data to "+to)
	return nil
}
