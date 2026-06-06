package app

import (
	"path/filepath"
	"testing"
)

func TestFilterSystemSdksDropsVfoxManagedAndErrorVersions(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	app.setVfoxHome(filepath.Join(tmp, "vfox-home"))
	userHome := filepath.Join(tmp, "user")
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)

	input := []SdkInfo{
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "vfox-home", "path-shims", "python.cmd"),
			Versions: []SdkVersion{{Version: "vfoxG: python for python is not available under C:\\vfox-home\\sdks\\python."}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(userHome, ".vfox", "sdks", "python", "python.exe"),
			Versions: []SdkVersion{{Version: "3.14.4"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "vfox-home", "cache", "python", "v-3.14.5", "python-3.14.5", "python.exe"),
			Versions: []SdkVersion{{Version: "3.14.5"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "WindowsApps", "python.exe"),
			Versions: []SdkVersion{{Version: "Python was not found; run without arguments to install from the Microsoft Store."}},
		},
		{
			Name:     "golang",
			Source:   "system",
			Path:     filepath.Join(tmp, "go", "bin", "go.exe"),
			Versions: []SdkVersion{{Version: "1.26.3"}},
		},
	}

	got := app.filterSystemSdks(input)
	if len(got) != 1 {
		t.Fatalf("got %d SDKs, want 1: %+v", len(got), got)
	}
	if got[0].Name != "golang" || got[0].Versions[0].Version != "1.26.3" {
		t.Fatalf("unexpected SDK left after filtering: %+v", got[0])
	}
}
