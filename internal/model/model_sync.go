package model

type SdkEnvironmentImportResult struct {
	Path               string   `json:"path"`
	ImportedCustomSdks int      `json:"importedCustomSdks"`
	SkippedCustomSdks  int      `json:"skippedCustomSdks"`
	VfoxSdksFound      int      `json:"vfoxSdksFound"`
	InstalledVfoxSdks  int      `json:"installedVfoxSdks"`
	SkippedVfoxSdks    int      `json:"skippedVfoxSdks"`
	Warnings           []string `json:"warnings"`
}
