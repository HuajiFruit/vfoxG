//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) windowsPathShimDir() string {
	return a.getVfoxHomePath("path-shims")
}

func (a *App) writeWindowsSDKShims(pluginName string) ([]string, error) {
	if strings.TrimSpace(pluginName) == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}
	shimDir := a.windowsPathShimDir()
	if shimDir == "" {
		return nil, fmt.Errorf("unable to resolve shim directory")
	}
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return nil, err
	}
	aliases := windowsSDKShimAliases(pluginName)
	sdkPath := a.getVfoxHomePath("sdks", pluginName)
	for _, alias := range aliases {
		shimName := windowsSafeShimName(alias) + ".cmd"
		shimPath := filepath.Join(shimDir, shimName)
		if err := os.WriteFile(shimPath, []byte(windowsShimScript(pluginName, alias, sdkPath)), 0644); err != nil {
			return nil, err
		}
	}
	return aliases, nil
}

func (a *App) removeWindowsSDKShims(pluginName string, aliases []string) error {
	shimDir := a.windowsPathShimDir()
	if shimDir == "" {
		return nil
	}
	if len(aliases) == 0 {
		aliases = windowsSDKShimAliases(pluginName)
	}
	for _, alias := range aliases {
		_ = os.Remove(filepath.Join(shimDir, windowsSafeShimName(alias)+".cmd"))
	}
	return nil
}
