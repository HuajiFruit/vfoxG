package app

// CheckVfoxInPath reports whether the vfox executable directory is in the user PATH.
func (a *App) CheckVfoxInPath() (bool, error) {
	return a.checkVfoxInPath()
}

// AddVfoxToPath adds the vfox executable directory to the user PATH.
func (a *App) AddVfoxToPath() error {
	return a.addVfoxToPath()
}

// RemoveVfoxFromPath removes the vfox executable directory from the user PATH.
func (a *App) RemoveVfoxFromPath() error {
	return a.removeVfoxFromPath()
}

// CheckPluginPathOverride reports whether a plugin has an active SDK PATH override.
func (a *App) CheckPluginPathOverride(pluginName string) bool {
	return a.checkPluginPathOverride(pluginName)
}

// CheckAnyPathOverride reports whether any SDK PATH override is active.
func (a *App) CheckAnyPathOverride() bool {
	return a.checkAnyPathOverride()
}

// HijackSystemPath enables the SDK PATH override for one plugin.
func (a *App) HijackSystemPath(name string, exePath string) error {
	return a.hijackSystemPath(name, exePath)
}

// RestoreSystemPath disables the SDK PATH override for one plugin.
func (a *App) RestoreSystemPath(name string) error {
	return a.restoreSystemPath(name)
}

// HijackPluginSystemPath enables the SDK PATH override using a stored custom SDK when present.
func (a *App) HijackPluginSystemPath(pluginName string) error {
	return a.hijackPluginSystemPath(pluginName)
}

// RestorePluginSystemPath disables the SDK PATH override for one plugin.
func (a *App) RestorePluginSystemPath(pluginName string) error {
	return a.restorePluginSystemPath(pluginName)
}

// CheckPluginWin11CompatMode reports whether a plugin has compatibility PATH metadata.
func (a *App) CheckPluginWin11CompatMode(pluginName string) bool {
	return a.checkPluginWin11CompatMode(pluginName)
}

// CheckWin11CompatMode reports whether any compatibility PATH metadata exists.
func (a *App) CheckWin11CompatMode() bool {
	return a.checkWin11CompatMode()
}
