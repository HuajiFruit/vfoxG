package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) exportCurrentEnvironmentSdks() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}

	snapshot := a.collectSdkEnvironmentExport(time.Now())
	defaultDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		defaultDir = home
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export SDK environment",
		DefaultDirectory: defaultDir,
		DefaultFilename:  fmt.Sprintf("vfoxG-sdk-environment-%s.txt", snapshot.GeneratedAt.Format("20060102-150405")),
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".txt") {
		path += ".txt"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(formatSdkEnvironmentExport(snapshot)), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) previewCurrentEnvironmentSdks() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}
	snapshot := a.collectSdkEnvironmentExport(time.Now())
	return formatSdkEnvironmentExport(snapshot), nil
}
