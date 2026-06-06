package model

type PlatformInfo struct {
	OS                  string `json:"os"`
	Name                string `json:"name"`
	CoreOS              string `json:"coreOS"`
	CoreArch            string `json:"coreArch"`
	DownloadPath        string `json:"downloadPath"`
	DefaultDownloadPath string `json:"defaultDownloadPath"`
	VfoxPathTarget      string `json:"vfoxPathTarget"`
	SDKPathTarget       string `json:"sdkPathTarget"`
	ShellProfile        string `json:"shellProfile"`
	RequiresElevation   bool   `json:"requiresElevation"`
	RestartHint         string `json:"restartHint"`
}
