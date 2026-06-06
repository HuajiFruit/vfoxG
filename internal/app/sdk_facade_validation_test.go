package app

import (
	"path/filepath"
	"testing"
)

func TestSDKApiRejectsEmptyInputs(t *testing.T) {
	app := NewApp()
	tmp := t.TempDir()
	app.setVfoxHome(filepath.Join(tmp, "home"))

	if _, err := app.UseVersion("", "1.0.0"); err == nil {
		t.Fatal("UseVersion should reject empty plugin name")
	}
	if _, err := app.UnuseVersion(""); err == nil {
		t.Fatal("UnuseVersion should reject empty plugin name")
	}
	if err := app.InstallVersion("python", ""); err == nil {
		t.Fatal("InstallVersion should reject empty version")
	}
	if err := app.UninstallVersion("", "1.0.0"); err == nil {
		t.Fatal("UninstallVersion should reject empty plugin name")
	}
	if _, err := app.GetVersionPath("python", ""); err == nil {
		t.Fatal("GetVersionPath should reject empty version")
	}
	if _, err := app.SearchVersions(""); err == nil {
		t.Fatal("SearchVersions should reject empty plugin name")
	}
	if err := app.AddNonVfoxSdk("", filepath.Join(tmp, "python"), "3.12.0"); err == nil {
		t.Fatal("AddNonVfoxSdk should reject empty plugin name")
	}
	if err := app.RemoveNonVfoxSdk("python", ""); err == nil {
		t.Fatal("RemoveNonVfoxSdk should reject empty path")
	}
	if _, err := app.UseCustomSdk("", filepath.Join(tmp, "python")); err == nil {
		t.Fatal("UseCustomSdk should reject empty plugin name")
	}
	if got := app.DetectSdkPathVersion("", filepath.Join(tmp, "python")); got != "unknown" {
		t.Fatalf("DetectSdkPathVersion empty name = %q, want unknown", got)
	}
	if _, err := app.GetActiveCustomSdk(""); err == nil {
		t.Fatal("GetActiveCustomSdk should reject empty plugin name")
	}
}
