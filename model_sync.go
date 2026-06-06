package main

import "time"

type sdkEnvironmentExport struct {
	GeneratedAt  time.Time
	Platform     PlatformInfo
	VfoxInPath   bool
	PathOverride bool
	VfoxSdks     []sdkEnvironmentVfoxSdk
	SystemSdks   []SdkInfo
	CustomSdks   map[string][]SdkInfo
	Warnings     []string
}

type sdkEnvironmentVfoxSdk struct {
	Name             string
	Versions         []SdkVersion
	Detail           SdkDetail
	VersionPaths     map[string]string
	ActiveCustomPath string
}

type SdkEnvironmentImportResult struct {
	Path               string   `json:"path"`
	ImportedCustomSdks int      `json:"importedCustomSdks"`
	SkippedCustomSdks  int      `json:"skippedCustomSdks"`
	VfoxSdksFound      int      `json:"vfoxSdksFound"`
	InstalledVfoxSdks  int      `json:"installedVfoxSdks"`
	SkippedVfoxSdks    int      `json:"skippedVfoxSdks"`
	Warnings           []string `json:"warnings"`
}

type sdkEnvironmentImportRow struct {
	Kind    string
	Name    string
	Version string
	Path    string
	Current bool
}
