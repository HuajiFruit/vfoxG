package app

import (
	"path/filepath"
	"testing"
)

func TestNormalizeDownloadPathExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := normalizeDownloadPath(filepath.Join("~", "vfox-data"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "vfox-data")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("normalizeDownloadPath() = %q, want %q", got, want)
	}
}
