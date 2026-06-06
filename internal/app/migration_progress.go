package app

import (
	"path/filepath"
	"strings"
	"time"
)

type migrationTracker struct {
	app       *App
	total     int
	completed int
	current   string
	startedAt time.Time
	oldRoot   string
}

func (m *migrationTracker) start(path string) {
	if m == nil {
		return
	}
	m.current = migrationDisplayPath(path, m.oldRoot)
	m.app.emitMigrationProgress("copying", m.current, m.completed, m.total, m.startedAt)
}

func (m *migrationTracker) finish(path string) {
	if m == nil {
		return
	}
	m.current = migrationDisplayPath(path, m.oldRoot)
	m.completed++
	m.app.emitMigrationProgress("copying", m.current, m.completed, m.total, m.startedAt)
}

func migrationDisplayPath(path string, root string) string {
	if rel, err := filepath.Rel(root, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Base(path)
}

func (a *App) emitMigrationProgress(stage string, current string, completed int, total int, startedAt time.Time) {
	percent := 0
	if total > 0 {
		percent = int(float64(completed) / float64(total) * 100)
		if percent > 100 {
			percent = 100
		}
	}
	estimatedRemaining := 0
	if completed > 0 && total > completed {
		elapsed := time.Since(startedAt).Seconds()
		perItem := elapsed / float64(completed)
		estimatedRemaining = int(perItem * float64(total-completed))
	}
	a.emitEvent("migration-progress", MigrationProgress{
		Stage:              stage,
		Current:            current,
		Completed:          completed,
		Total:              total,
		Percent:            percent,
		EstimatedRemaining: estimatedRemaining,
	})
}
