package app

// GetDownloadPathInfo returns the active download directory and migration status.
func (a *App) GetDownloadPathInfo() (DownloadPathInfo, error) {
	return a.getDownloadPathInfo()
}

// SetDownloadPath changes the download directory without copying existing SDK data.
func (a *App) SetDownloadPath(path string) (DownloadPathInfo, error) {
	return a.setDownloadPath(path)
}

// SetDownloadPathWithMigration changes the download directory and optionally copies existing SDK data.
func (a *App) SetDownloadPathWithMigration(path string, migrateVfoxData bool) (DownloadPathInfo, error) {
	return a.setDownloadPathWithMigration(path, migrateVfoxData)
}

// ResetDownloadPath restores the default download directory without copying existing SDK data.
func (a *App) ResetDownloadPath() (DownloadPathInfo, error) {
	return a.resetDownloadPath()
}

// ResetDownloadPathWithMigration restores the default download directory and optionally copies SDK data.
func (a *App) ResetDownloadPathWithMigration(migrateVfoxData bool) (DownloadPathInfo, error) {
	return a.resetDownloadPathWithMigration(migrateVfoxData)
}

// SelectDownloadPath opens a directory picker for the SDK download location.
func (a *App) SelectDownloadPath() (string, error) {
	return a.selectDownloadPath()
}

// GetPlatformInfo returns OS-specific PATH and download-directory metadata.
func (a *App) GetPlatformInfo() PlatformInfo {
	return a.getPlatformInfo()
}
