package app

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCleanedEnvForVfoxRemovesManagedSdkAndShimPaths(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	t.Setenv("VFOX_HOME", tmp)
	userHome := filepath.Join(tmp, "user")
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)

	vfoxSdksDir := filepath.Join(tmp, "sdks")
	vfoxShimDir := filepath.Join(tmp, "path-shims")
	vfoxCacheDir := filepath.Join(tmp, "cache")
	legacySdksDir := filepath.Join(userHome, ".vfox", "sdks")
	legacyCacheDir := filepath.Join(userHome, ".vfox", "cache")
	paths := []string{
		filepath.Join(vfoxSdksDir, "python"),
		vfoxShimDir,
		filepath.Join(vfoxCacheDir, "python", "v-3.14.5", "python-3.14.5"),
		filepath.Join(legacySdksDir, "python", "Scripts"),
		filepath.Join(legacyCacheDir, "nodejs", "v-22.0.0", "node-v22.0.0", "bin"),
		filepath.Join(tmp, "sdks-other"),
		filepath.Join(tmp, "tools"),
	}
	t.Setenv("PATH", strings.Join(paths, string(filepath.ListSeparator)))

	env := app.getCleanedEnvForVfox()
	var gotPath string
	var gotVfoxHome string
	for _, e := range env {
		if strings.HasPrefix(strings.ToLower(e), "path=") {
			gotPath = e[5:]
		}
		if strings.HasPrefix(strings.ToLower(e), "vfox_home=") {
			gotVfoxHome = e[len("VFOX_HOME="):]
		}
	}
	if gotPath == "" {
		t.Fatal("PATH not found in cleaned environment")
	}
	if filepath.Clean(gotVfoxHome) != filepath.Clean(tmp) {
		t.Fatalf("VFOX_HOME = %q, want %q", gotVfoxHome, tmp)
	}

	gotParts := filepath.SplitList(gotPath)
	if len(gotParts) != 2 {
		t.Fatalf("got %d PATH entries, want 2: %v", len(gotParts), gotParts)
	}
	if filepath.Clean(gotParts[0]) != filepath.Clean(filepath.Join(tmp, "sdks-other")) {
		t.Fatalf("unexpected first PATH entry: %q", gotParts[0])
	}
	if filepath.Clean(gotParts[1]) != filepath.Clean(filepath.Join(tmp, "tools")) {
		t.Fatalf("unexpected second PATH entry: %q", gotParts[1])
	}
}
