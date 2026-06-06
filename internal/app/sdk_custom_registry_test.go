package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCustomSdksToKeep(t *testing.T) {
	sdks := []SdkInfo{
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join("tmp", "python-a", "bin", "python"),
			Versions: []SdkVersion{{Version: "3.12.0"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join("tmp", "python-b", "bin", "python"),
			Versions: []SdkVersion{{Version: "3.13.0"}},
		},
	}

	got, err := customSdksToKeep(sdks, "")
	if err != nil {
		t.Fatalf("empty keep path returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("empty keep path = %+v, want nil", got)
	}

	got, err = customSdksToKeep(sdks, sdks[1].Path)
	if err != nil {
		t.Fatalf("matching keep path returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d SDKs, want 1", len(got))
	}
	if got[0].Path != sdks[1].Path || got[0].Versions[0].Version != "3.13.0" {
		t.Fatalf("kept SDK mismatch: %+v", got[0])
	}

	_, err = customSdksToKeep(sdks, filepath.Join("tmp", "missing", "bin", "python"))
	if err == nil {
		t.Fatal("missing keep path should return an error")
	}
}

func TestRemoveNonVfoxSdkUsesNormalizedPath(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	app.setVfoxHome(filepath.Join(tmp, "home"))

	exeDir := filepath.Join(tmp, "sdk", "bin")
	if err := os.MkdirAll(exeDir, 0755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(exeDir, "python")
	if err := os.WriteFile(exePath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := app.AddNonVfoxSdk("python", exePath, "3.12.0"); err != nil {
		t.Fatal(err)
	}

	equivalentPath := exeDir + string(filepath.Separator) + "." + string(filepath.Separator) + "python"
	if err := app.RemoveNonVfoxSdk("python", equivalentPath); err != nil {
		t.Fatal(err)
	}

	if got := app.GetNonVfoxSdksMap()["python"]; len(got) != 0 {
		t.Fatalf("custom SDK was not removed: %+v", got)
	}
}
