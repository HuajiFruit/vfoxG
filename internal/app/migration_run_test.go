package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSetDownloadPathWithMigrationCopiesVfoxData(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	current := filepath.Join(tmp, "current")
	target := filepath.Join(tmp, "target")
	prepareMigratableVfoxHome(t, current)

	app := NewApp()
	app.setVfoxHome(current)

	info, err := app.SetDownloadPathWithMigration(target, true)
	if err != nil {
		t.Fatal(err)
	}
	assertMigratedDownloadPath(t, info, configDir, target)
}

func prepareMigratableVfoxHome(t *testing.T, current string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(current, "cache", "python"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(current, "plugin", "python"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(current, "path-shims"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, ".vfox.toml"), []byte("python = \"3.13.13\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "cache", "python", "version.txt"), []byte("3.13.13"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "plugin", "python", "metadata.lua"), []byte("metadata"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "gui-plugins-cache.json"), []byte(`{"plugins":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "gui-non-vfox-sdks.json"), []byte(`{"python":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "hijacked_paths.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "path-shims", "python.cmd"), []byte("@echo off\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertMigratedDownloadPath(t *testing.T, info DownloadPathInfo, configDir string, target string) {
	t.Helper()

	if !samePath(info.Path, target) {
		t.Fatalf("path = %q, want %q", info.Path, target)
	}
	if !info.HasMigratableData {
		t.Fatal("new target should report migratable data after copy")
	}

	for _, path := range []string{
		filepath.Join(target, ".vfox.toml"),
		filepath.Join(target, "cache", "python", "version.txt"),
		filepath.Join(target, "plugin", "python", "metadata.lua"),
		filepath.Join(target, "gui-plugins-cache.json"),
		filepath.Join(target, "gui-non-vfox-sdks.json"),
		filepath.Join(target, "hijacked_paths.json"),
		filepath.Join(target, "path-shims", "python.cmd"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected migrated path %s: %v", path, err)
		}
	}

	configPath := filepath.Join(configDir, "vfoxG", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !samePath(config.VfoxHome, target) {
		t.Fatalf("saved VfoxHome = %q, want %q", config.VfoxHome, target)
	}
}

func TestSetDownloadPathWithMigrationRejectsExistingDestinationEntry(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	current := filepath.Join(tmp, "current")
	target := filepath.Join(tmp, "target")
	if err := os.MkdirAll(filepath.Join(current, "cache", "python"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "cache", "python", "version.txt"), []byte("3.13.13"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "cache"), 0755); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.setVfoxHome(current)

	if _, err := app.SetDownloadPathWithMigration(target, true); err == nil {
		t.Fatal("expected existing destination entry error")
	}
	if !samePath(app.getVfoxHome(), current) {
		t.Fatalf("vfox home changed after failed migration: %s", app.getVfoxHome())
	}
}
