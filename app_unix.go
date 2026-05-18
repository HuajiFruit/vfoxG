//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func hideWindow(cmd *exec.Cmd) {
	// Not needed on Unix
}

func getVfoxExeName() string {
	return "vfox"
}

func (a *App) ensureJunction(linkPath string, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			existing, readErr := os.Readlink(linkPath)
			if readErr == nil && existing == target {
				return nil
			}
		} else if fi.IsDir() {
			if linkPath == target {
				return nil
			}
		}
	}
	_ = os.RemoveAll(linkPath)
	_ = os.MkdirAll(filepath.Dir(linkPath), 0755)

	return os.Symlink(target, linkPath)
}

func (a *App) CheckVfoxInPath() (bool, error) {
	// Dummy implementation for Unix
	return false, nil
}

func (a *App) AddVfoxToPath() error {
	return nil
}

func (a *App) RemoveVfoxFromPath() error {
	return nil
}

func (a *App) HijackSystemPath(name string, exePath string) error {
	return nil
}

func (a *App) RestoreSystemPath(name string) error {
	return nil
}

func (a *App) HijackPluginSystemPath(pluginName string) error {
	return nil
}

func (a *App) RestorePluginSystemPath(pluginName string) error {
	return nil
}

func (a *App) CheckPluginWin11CompatMode(pluginName string) bool {
	return false
}

func (a *App) CheckWin11CompatMode() bool {
	return false
}

func findExecutable(exe string, cleanEnv []string) string {
	lookCmd := exec.Command("which", exe)
	lookCmd.Env = cleanEnv
	whereOut, err := lookCmd.Output()
	if err != nil {
		return ""
	}
	exePath := strings.TrimSpace(string(whereOut))
	return exePath
}
