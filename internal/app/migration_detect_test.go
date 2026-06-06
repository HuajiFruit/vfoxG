package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasMigratableVfoxHomeData(t *testing.T) {
	tmp := t.TempDir()
	if ok, err := hasMigratableVfoxHomeData(tmp); err != nil || ok {
		t.Fatalf("empty home migratable = %v, %v; want false, nil", ok, err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "sdks"), 0755); err != nil {
		t.Fatal(err)
	}
	if ok, err := hasMigratableVfoxHomeData(tmp); err != nil || ok {
		t.Fatalf("empty sdks dir migratable = %v, %v; want false, nil", ok, err)
	}

	if err := os.WriteFile(filepath.Join(tmp, "sdks", "python"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if ok, err := hasMigratableVfoxHomeData(tmp); err != nil || !ok {
		t.Fatalf("sdk data migratable = %v, %v; want true, nil", ok, err)
	}
}
