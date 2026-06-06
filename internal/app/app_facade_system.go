package app

// GetCachedSystemSdks returns a copy of the cached system SDK list.
func (a *App) GetCachedSystemSdks() []SdkInfo {
	return a.getCachedSystemSdks()
}

// ScanSystemSdks scans system SDKs concurrently and updates the GUI cache.
func (a *App) ScanSystemSdks() {
	a.scanSystemSdks()
}
