package app

// ExportCurrentEnvironmentSdks writes the currently detected SDK environment to a text file.
func (a *App) ExportCurrentEnvironmentSdks() (string, error) {
	return a.exportCurrentEnvironmentSdks()
}

// PreviewCurrentEnvironmentSdks returns the same text that would be exported.
func (a *App) PreviewCurrentEnvironmentSdks() (string, error) {
	return a.previewCurrentEnvironmentSdks()
}

// ImportSdkEnvironmentFromTxt imports SDK references from a selected text export.
func (a *App) ImportSdkEnvironmentFromTxt() (SdkEnvironmentImportResult, error) {
	return a.importSdkEnvironmentFromTxt()
}
