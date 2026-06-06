package main

import "strings"

func parseSdkDetailOutput(sdkName string, currentVersion string, commandOutput string) SdkDetail {
	currentVersion = normalizeSdkVersion(currentVersion)
	lines := strings.Split(commandOutput, "\n")
	detail := SdkDetail{Name: sdkName, Current: currentVersion}
	markerCurrentCount := 0

	for _, line := range lines {
		version, markerCurrent, ok := parseSdkDetailVersionLine(line)
		if !ok {
			continue
		}

		isCurrent := false
		if currentVersion != "" {
			isCurrent = sameSdkVersion(version, currentVersion)
		} else if markerCurrent {
			isCurrent = true
			detail.Current = version
			markerCurrentCount++
		}

		detail.Versions = append(detail.Versions, SdkVersionDetail{
			Version:   version,
			IsCurrent: isCurrent,
		})
	}

	// Some vfox outputs can leave more than one line marked current. Do not expose
	// multiple active versions to the UI when `vfox current` did not confirm one.
	if currentVersion == "" && markerCurrentCount > 1 {
		detail.Current = ""
		for i := range detail.Versions {
			detail.Versions[i].IsCurrent = false
		}
	}

	return detail
}

func parseSdkDetailVersionLine(line string) (version string, isCurrent bool, ok bool) {
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	if line == "" {
		return "", false, false
	}

	if strings.HasPrefix(line, "->") {
		isCurrent = true
		line = strings.TrimSpace(strings.TrimPrefix(line, "->"))
	}

	if versionWithoutMarker, foundMarker := trimSdkCurrentVersionMarker(line); foundMarker {
		isCurrent = true
		line = versionWithoutMarker
	}

	if line == "" || strings.Contains(line, "installed sdk") || strings.HasPrefix(line, "custom-sys-") {
		return "", false, false
	}
	return line, isCurrent, true
}
