package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorePlatformNames(t *testing.T) {
	if coreOSName() == "" {
		t.Fatal("core OS name must not be empty")
	}
	if coreArchName() == "" {
		t.Fatal("core arch name must not be empty")
	}
}

func TestGetVfoxExecutableFromWorkingDirectoryCore(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(tmp, "core", coreOSName(), coreArchName())
	if err := os.MkdirAll(expectedDir, 0755); err != nil {
		t.Fatal(err)
	}
	expectedExe := filepath.Join(expectedDir, getVfoxExeName())
	if err := os.WriteFile(expectedExe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	coreDir := app.getCoreDir()
	if filepath.Clean(coreDir) != filepath.Clean(expectedDir) {
		t.Skipf("core dir resolved outside test fixture: %s", coreDir)
	}

	got, err := app.getVfoxExecutable()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(expectedExe) {
		t.Fatalf("got %q, want %q", got, expectedExe)
	}
}

func TestGetVfoxExecutableReportsMissingCore(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	coreDir := app.getCoreDir()
	if !strings.HasPrefix(filepath.Clean(coreDir), filepath.Clean(tmp)) {
		t.Skipf("core dir resolved outside test fixture: %s", coreDir)
	}

	_, err = app.getVfoxExecutable()
	if err == nil {
		t.Fatal("expected missing core executable error")
	}
	if !strings.Contains(err.Error(), "vfox core executable not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
