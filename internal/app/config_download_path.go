package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) getDownloadPathInfo() (DownloadPathInfo, error) {
	path := a.getVfoxHome()
	defaultPath := a.defaultVfoxHome()
	hasMigratableData, err := hasMigratableVfoxHomeData(path)
	if err != nil {
		return DownloadPathInfo{}, err
	}
	return DownloadPathInfo{
		Path:              path,
		DefaultPath:       defaultPath,
		IsDefault:         samePath(path, defaultPath),
		HasMigratableData: hasMigratableData,
	}, nil
}

func (a *App) setDownloadPath(path string) (DownloadPathInfo, error) {
	return a.setDownloadPathWithMigration(path, false)
}

func (a *App) setDownloadPathWithMigration(path string, migrateVfoxData bool) (DownloadPathInfo, error) {
	normalized, err := normalizeDownloadPath(path)
	if err != nil {
		return DownloadPathInfo{}, err
	}
	current := a.getVfoxHome()
	if samePath(current, normalized) {
		return a.getDownloadPathInfo()
	}
	if migrateVfoxData {
		if err := a.migrateVfoxHomeData(current, normalized); err != nil {
			return DownloadPathInfo{}, err
		}
	}
	if err := os.MkdirAll(normalized, 0755); err != nil {
		return DownloadPathInfo{}, err
	}
	if err := a.saveAppConfig(AppConfig{VfoxHome: normalized}); err != nil {
		return DownloadPathInfo{}, err
	}
	a.setVfoxHome(normalized)
	if migrateVfoxData {
		a.repairMigratedSdkEntrypoints(current)
	}
	a.emitEvent("vfox-log", "[INFO] VFOX_HOME="+normalized)
	a.emitEvent("vfox-home-changed")
	a.emitEvent("sdk-list-changed")
	go a.refreshAvailablePlugins()
	go a.scanSystemSdks()
	return a.getDownloadPathInfo()
}

func (a *App) resetDownloadPath() (DownloadPathInfo, error) {
	return a.resetDownloadPathWithMigration(false)
}

func (a *App) resetDownloadPathWithMigration(migrateVfoxData bool) (DownloadPathInfo, error) {
	defaultPath := a.defaultVfoxHome()
	current := a.getVfoxHome()
	if samePath(current, defaultPath) {
		return a.getDownloadPathInfo()
	}
	if migrateVfoxData {
		if err := a.migrateVfoxHomeData(current, defaultPath); err != nil {
			return DownloadPathInfo{}, err
		}
	}
	if err := os.MkdirAll(defaultPath, 0755); err != nil {
		return DownloadPathInfo{}, err
	}
	if err := a.saveAppConfig(AppConfig{}); err != nil {
		return DownloadPathInfo{}, err
	}
	a.setVfoxHome(defaultPath)
	if migrateVfoxData {
		a.repairMigratedSdkEntrypoints(current)
	}
	a.emitEvent("vfox-log", "[INFO] VFOX_HOME="+defaultPath)
	a.emitEvent("vfox-home-changed")
	a.emitEvent("sdk-list-changed")
	go a.refreshAvailablePlugins()
	go a.scanSystemSdks()
	return a.getDownloadPathInfo()
}

func (a *App) selectDownloadPath() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}
	current := a.getVfoxHome()
	defaultDir := current
	if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
		defaultDir = filepath.Dir(defaultDir)
		if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
			defaultDir = a.appInstallDir()
		}
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select SDK and plugin download directory",
		DefaultDirectory:     defaultDir,
		CanCreateDirectories: true,
	})
}
