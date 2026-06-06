package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func customSdksToKeep(list []SdkInfo, keepPath string) ([]SdkInfo, error) {
	keepPath = strings.TrimSpace(keepPath)
	if keepPath == "" {
		return nil, nil
	}
	for _, sdk := range list {
		if samePath(sdk.Path, keepPath) {
			return []SdkInfo{sdk}, nil
		}
	}
	return nil, fmt.Errorf("custom SDK path is not registered: %s", keepPath)
}

func (a *App) getNonVfoxSdksFile() string {
	return a.getVfoxHomePath("gui-non-vfox-sdks.json")
}

func (a *App) getNonVfoxSdksMap() map[string][]SdkInfo {
	filePath := a.getNonVfoxSdksFile()
	data, err := os.ReadFile(filePath)
	result := make(map[string][]SdkInfo)
	if err == nil {
		_ = json.Unmarshal(data, &result)
	}
	return result
}

func (a *App) saveNonVfoxSdksMap(customSdks map[string][]SdkInfo) error {
	return a.writeJSONFile(a.getNonVfoxSdksFile(), customSdks)
}

func (a *App) getNonVfoxSdks() map[string][]SdkInfo {
	return a.getNonVfoxSdksMap()
}

func (a *App) addNonVfoxSdk(name string, exePath string, version string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	exePath = strings.TrimSpace(exePath)
	if err := validateSDKExecutablePath(exePath); err != nil {
		return err
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}

	if version == "" {
		version = "unknown"
	}

	customSdks := a.getNonVfoxSdksMap()
	list := customSdks[name]
	for _, existing := range list {
		if samePath(existing.Path, exePath) {
			return fmt.Errorf("path already exists")
		}
	}
	customSdks[name] = append(customSdks[name], SdkInfo{
		Name:     name,
		Source:   "system",
		Path:     exePath,
		Versions: []SdkVersion{{Version: version}},
	})
	return a.saveNonVfoxSdksMap(customSdks)
}

func (a *App) removeNonVfoxSdk(name string, exePath string) error {
	name = strings.TrimSpace(name)
	exePath = strings.TrimSpace(exePath)
	if name == "" || exePath == "" {
		return fmt.Errorf("plugin name and path cannot be empty")
	}
	customSdks := a.getNonVfoxSdksMap()
	list := customSdks[name]
	var newList []SdkInfo
	for _, existing := range list {
		if !samePath(existing.Path, exePath) {
			newList = append(newList, existing)
			continue
		}

		activePath, _ := a.getActiveCustomSdk(name)
		if samePath(activePath, exePath) {
			sdkLinkPath := a.getVfoxHomePath("sdks", name)
			a.removeJunctionIfExists(sdkLinkPath)
		}
	}
	if len(newList) == len(list) {
		return fmt.Errorf("path not found")
	}
	if len(newList) == 0 {
		delete(customSdks, name)
	} else {
		customSdks[name] = newList
	}
	return a.saveNonVfoxSdksMap(customSdks)
}
