package app

import (
	"path/filepath"
	"testing"
)

func TestGetVfoxHomeUsesConfiguredPathBeforeEnvironment(t *testing.T) {
	app := NewApp()
	configured := filepath.Join(t.TempDir(), "configured")
	envHome := filepath.Join(t.TempDir(), "env")
	t.Setenv("VFOX_HOME", envHome)

	app.setVfoxHome(configured)

	if got := app.getVfoxHome(); filepath.Clean(got) != filepath.Clean(configured) {
		t.Fatalf("getVfoxHome() = %q, want configured path %q", got, configured)
	}
}

func TestDefaultVfoxHomeUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	got := NewApp().defaultVfoxHome()
	want := filepath.Join(configDir, "vfoxG", "vfox-home")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("defaultVfoxHome() = %q, want %q", got, want)
	}
}

func TestDefaultUserVfoxHomeUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	got, err := defaultUserVfoxHome()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configDir, "vfoxG", "vfox-home")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("defaultUserVfoxHome() = %q, want %q", got, want)
	}
}
