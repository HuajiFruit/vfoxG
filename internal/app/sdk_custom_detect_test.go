package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSDKExecutablePath(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "sdk-exe")
	if err := os.WriteFile(exe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "empty", path: "", wantErr: true},
		{name: "missing", path: filepath.Join(tmp, "missing"), wantErr: true},
		{name: "directory", path: tmp, wantErr: true},
		{name: "file", path: exe, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSDKExecutablePath(tt.path)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
