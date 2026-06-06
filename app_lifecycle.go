package main

import (
	"context"
	"os"
	"path/filepath"
	stdruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// NewApp creates the Wails application state.
func NewApp() *App {
	return &App{}
}

// startup stores the Wails context and warms the background caches.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.loadVfoxHomeSetting(); err != nil {
		a.emitEvent("vfox-log", "[APP ERROR] "+err.Error())
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		a.emitEvent("vfox-log", "[APP ERROR] "+err.Error())
	}
	go a.scanSystemSdks()
	go a.refreshAvailablePlugins()
}

func (a *App) emitEvent(name string, data ...interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data...)
}

func (a *App) appInstallDir() string {
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		if stdruntime.GOOS == "darwin" && filepath.Base(exeDir) == "MacOS" {
			contentsDir := filepath.Dir(exeDir)
			if filepath.Base(contentsDir) == "Contents" {
				return filepath.Dir(filepath.Dir(contentsDir))
			}
		}
		return exeDir
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return "."
}
