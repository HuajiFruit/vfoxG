package app

// GetInstalledSdks returns SDKs reported by `vfox ls`.
func (a *App) GetInstalledSdks() ([]SdkInfo, error) {
	return a.getInstalledSdks()
}

// GetAllSdks merges live vfox SDKs with cached system SDKs.
func (a *App) GetAllSdks() ([]SdkInfo, error) {
	return a.getAllSdks()
}

// GetSdkDetail returns installed versions and current selection for one SDK.
func (a *App) GetSdkDetail(name string) (SdkDetail, error) {
	return a.getSdkDetail(name)
}

// GetVersionPath returns the absolute install path for one SDK version.
func (a *App) GetVersionPath(name string, version string) (string, error) {
	return a.getVersionPath(name, version)
}

// SearchVersions searches available versions for one SDK plugin.
func (a *App) SearchVersions(name string) ([]string, error) {
	return a.searchVersions(name)
}

// InstallVersion installs a version and streams progress through vfox-log events.
func (a *App) InstallVersion(name string, version string) error {
	return a.installVersion(name, version)
}

// UninstallVersion removes one installed SDK version.
func (a *App) UninstallVersion(name string, version string) error {
	return a.uninstallVersion(name, version)
}

// UseVersion switches the global vfox SDK version and refreshes managed entrypoints.
func (a *App) UseVersion(name string, version string) (string, error) {
	return a.useVersion(name, version)
}

// UnuseVersion clears the global vfox selection and removes GUI-managed links.
func (a *App) UnuseVersion(name string) (string, error) {
	return a.unuseVersion(name)
}

// DetectSdkPathVersion tries to extract a version from a custom SDK executable.
func (a *App) DetectSdkPathVersion(name string, exePath string) string {
	return a.detectSdkPathVersion(name, exePath)
}

// GetNonVfoxSdksMap returns the persisted map of custom SDKs.
func (a *App) GetNonVfoxSdksMap() map[string][]SdkInfo {
	return a.getNonVfoxSdksMap()
}

// GetNonVfoxSdks exposes the full non-vfox SDK list to the UI.
func (a *App) GetNonVfoxSdks() map[string][]SdkInfo {
	return a.getNonVfoxSdks()
}

// AddNonVfoxSdk registers a custom SDK executable path.
func (a *App) AddNonVfoxSdk(name string, exePath string, version string) error {
	return a.addNonVfoxSdk(name, exePath, version)
}

// RemoveNonVfoxSdk removes a custom SDK executable path.
func (a *App) RemoveNonVfoxSdk(name string, exePath string) error {
	return a.removeNonVfoxSdk(name, exePath)
}

// UseCustomSdk activates a custom SDK path without calling `vfox use`.
func (a *App) UseCustomSdk(name string, exePath string) (string, error) {
	return a.useCustomSdk(name, exePath)
}

// GetActiveCustomSdk returns the active custom SDK path when it is outside VFOX_HOME.
func (a *App) GetActiveCustomSdk(name string) (string, error) {
	return a.getActiveCustomSdk(name)
}
