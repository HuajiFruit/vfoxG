package app

import (
	"encoding/json"
	"os"

	"vfoxG/internal/parser"
)

func (a *App) getAvailablePlugins() ([]PluginInfo, error) {
	cacheFile := a.getCacheFile()
	var plugins []PluginInfo

	data, err := os.ReadFile(cacheFile)
	if err == nil {
		if json.Unmarshal(data, &plugins) == nil && len(plugins) > 0 {
			return a.applyIsAddedStatus(plugins), nil
		}
	}

	return a.refreshAvailablePlugins()
}

func (a *App) refreshAvailablePlugins() ([]PluginInfo, error) {
	out, err := a.runVfoxCommandWithLock("available")
	if err != nil {
		return nil, err
	}

	plugins := parser.AvailablePlugins(out)

	if len(plugins) > 0 {
		_ = a.writeJSONFile(a.getCacheFile(), plugins)
	}

	return a.applyIsAddedStatus(plugins), nil
}
