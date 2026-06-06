package app

func (a *App) checkPluginPathOverride(pluginName string) bool {
	return a.checkPluginWin11CompatMode(pluginName)
}

func (a *App) checkAnyPathOverride() bool {
	return a.checkWin11CompatMode()
}
