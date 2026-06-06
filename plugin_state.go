package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) getAddedPlugins() ([]string, error) {
	vfoxHome := a.getVfoxHome()
	if strings.TrimSpace(vfoxHome) == "" {
		return []string{}, fmt.Errorf("unable to resolve vfox home directory")
	}
	pluginDir := filepath.Join(vfoxHome, "plugin")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			plugins = append(plugins, entry.Name())
		}
	}
	return plugins, nil
}

// applyIsAddedStatus marks market plugins that already have vfox-managed SDKs.
func (a *App) applyIsAddedStatus(plugins []PluginInfo) []PluginInfo {
	installedSdks, _ := a.getInstalledSdks()
	addedMap := make(map[string]bool)
	for _, sdk := range installedSdks {
		if sdk.Source == "vfox" {
			addedMap[sdk.Name] = true
		}
	}
	for i := range plugins {
		plugins[i].IsAdded = addedMap[plugins[i].Name]
	}
	return plugins
}
