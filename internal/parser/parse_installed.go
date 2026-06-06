package parser

import (
	"regexp"
	"strings"

	"vfoxG/internal/model"
)

var (
	installedSdkPluginNameRegex = regexp.MustCompile(`^[├└]─[┬─](.+)`)
	installedSdkVersionRegex    = regexp.MustCompile(`^[│ ]\s*[├└]──(.*)`)
)

func InstalledSdks(commandOutput string) []model.SdkInfo {
	lines := strings.Split(commandOutput, "\n")
	installedSdks := make([]model.SdkInfo, 0)
	var currentSdk *model.SdkInfo

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if match := installedSdkPluginNameRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				installedSdks = append(installedSdks, *currentSdk)
			}
			currentSdk = &model.SdkInfo{Name: match[1], Versions: []model.SdkVersion{}, Source: "vfox"}
			continue
		}

		if match := installedSdkVersionRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk == nil {
				continue
			}
			sdkVersion := match[1]
			if !strings.HasPrefix(sdkVersion, "custom-sys-") {
				currentSdk.Versions = append(currentSdk.Versions, model.SdkVersion{Version: sdkVersion})
			}
		}
	}

	if currentSdk != nil {
		installedSdks = append(installedSdks, *currentSdk)
	}

	return installedSdks
}
