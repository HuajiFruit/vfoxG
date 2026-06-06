package main

import (
	"regexp"
	"strings"
)

var (
	installedSdkPluginNameRegex = regexp.MustCompile(`^[├└]─[┬─](.+)`)
	installedSdkVersionRegex    = regexp.MustCompile(`^[│ ]\s*[├└]──(.*)`)
)

func parseInstalledSdksOutput(commandOutput string) []SdkInfo {
	lines := strings.Split(commandOutput, "\n")
	installedSdks := make([]SdkInfo, 0)
	var currentSdk *SdkInfo

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if match := installedSdkPluginNameRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				installedSdks = append(installedSdks, *currentSdk)
			}
			currentSdk = &SdkInfo{Name: match[1], Versions: []SdkVersion{}, Source: "vfox"}
			continue
		}

		if match := installedSdkVersionRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk == nil {
				continue
			}
			sdkVersion := match[1]
			if !strings.HasPrefix(sdkVersion, "custom-sys-") {
				currentSdk.Versions = append(currentSdk.Versions, SdkVersion{Version: sdkVersion})
			}
		}
	}

	if currentSdk != nil {
		installedSdks = append(installedSdks, *currentSdk)
	}

	return installedSdks
}
