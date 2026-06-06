package main

import (
	"fmt"
	"os"
	"strings"
)

func (a *App) getSdkDetail(name string) (SdkDetail, error) {
	activeCustomPath, _ := a.getActiveCustomSdk(name)
	hasActiveCustom := activeCustomPath != ""
	currentOut, _ := a.runVfoxCommandWithLock("current", name)
	currentVersion := parseCurrentSdkVersion(name, currentOut)
	if hasActiveCustom {
		currentVersion = ""
	}

	// Fallback: if vfox current fails, read from .vfox.toml directly only
	// when there is no custom SDK junction active for this plugin.
	allowConfigFallback := true
	if currentVersion == "" || strings.Contains(currentVersion, "no current") {
		if hasActiveCustom {
			allowConfigFallback = false
		}
	}
	if allowConfigFallback && (currentVersion == "" || strings.Contains(currentVersion, "no current")) {
		if data, err := os.ReadFile(a.getVfoxHomePath(".vfox.toml")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, name+" ") || strings.HasPrefix(trimmed, name+"=") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						versionValue := strings.TrimSpace(parts[1])
						versionValue = strings.Trim(versionValue, "\"")
						currentVersion = normalizeSdkVersion(versionValue)
					}
					break
				}
			}
		}
	}

	if strings.Contains(currentVersion, "no current") {
		currentVersion = ""
	}

	out, err := a.runVfoxCommandWithLock("ls", name)
	if err != nil {
		// Keep the current marker when vfox cannot list versions for this plugin.
		return SdkDetail{Name: name, Versions: make([]SdkVersionDetail, 0), Current: currentVersion}, nil
	}

	detail := parseSdkDetailOutput(name, currentVersion, out)
	if hasActiveCustom {
		detail.Current = ""
		for i := range detail.Versions {
			detail.Versions[i].IsCurrent = false
		}
	}
	return detail, nil
}

func (a *App) readVfoxGlobalSelections() (map[string]string, error) {
	result := make(map[string]string)
	configPath := a.getVfoxHomePath(".vfox.toml")
	if strings.TrimSpace(configPath) == "" {
		return result, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		name, rawVersion, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		version := strings.TrimSpace(rawVersion)
		version = strings.Trim(version, `"'`)
		version = normalizeSdkVersion(version)
		if name == "" || isUnknownSdkVersion(version) || !containsDigit(version) {
			continue
		}
		result[name] = version
	}
	return result, nil
}

func containsDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func validateSdkNameAndVersion(name string, version string) (string, string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return "", "", fmt.Errorf("plugin name and version cannot be empty")
	}
	return name, version, nil
}
