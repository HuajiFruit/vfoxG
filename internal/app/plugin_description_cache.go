package app

// getCacheFile returns the GUI plugin market cache path.
func (a *App) getCacheFile() string {
	return a.getVfoxHomePath("gui-plugins-cache.json")
}
