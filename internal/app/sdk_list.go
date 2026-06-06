package app

import "vfoxG/internal/parser"

func (a *App) getInstalledSdks() ([]SdkInfo, error) {
	return a.getInstalledSdksUnlocked()
}

func (a *App) getInstalledSdksUnlocked() ([]SdkInfo, error) {
	out, err := a.runVfoxCommand("ls")
	if err != nil {
		return nil, err
	}
	return parser.InstalledSdks(out), nil
}

func (a *App) getAllSdks() ([]SdkInfo, error) {
	vfoxSdks, err := a.getInstalledSdks()
	if err != nil {
		vfoxSdks = []SdkInfo{}
	}

	cached := a.getCachedSystemSdks()
	seen := make(map[string]bool)
	result := make([]SdkInfo, 0, len(vfoxSdks)+len(cached))
	for _, sdk := range vfoxSdks {
		seen[sdk.Name] = true
		result = append(result, sdk)
	}
	for _, sdk := range cached {
		if !seen[sdk.Name] {
			result = append(result, sdk)
		}
	}
	return result, nil
}
