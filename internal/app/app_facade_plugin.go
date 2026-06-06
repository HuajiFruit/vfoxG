package app

// GetAvailablePlugins returns all available plugins, preferring the GUI cache.
func (a *App) GetAvailablePlugins() ([]PluginInfo, error) {
	return a.getAvailablePlugins()
}

// RefreshAvailablePlugins runs `vfox available` and refreshes the GUI cache.
func (a *App) RefreshAvailablePlugins() ([]PluginInfo, error) {
	return a.refreshAvailablePlugins()
}

// GetAddedPlugins returns plugins installed under the vfox plugin directory.
func (a *App) GetAddedPlugins() ([]string, error) {
	return a.getAddedPlugins()
}

// RemovePlugin removes a plugin and its managed SDK data.
func (a *App) RemovePlugin(name string) error {
	return a.removePlugin(name)
}

// RemovePluginWithOptions removes a plugin and optionally keeps one custom SDK path active.
func (a *App) RemovePluginWithOptions(name string, keepCustomSdkPath string) error {
	return a.removePluginWithOptions(name, keepCustomSdkPath)
}
