package parser

import "strings"

func CurrentSdkVersion(sdkName string, commandOutput string) string {
	sdkName = strings.TrimSpace(sdkName)
	for _, rawLine := range strings.Split(commandOutput, "\n") {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" {
			continue
		}

		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "no current") ||
			strings.Contains(lowerLine, "not installed") ||
			strings.Contains(lowerLine, "not supported") {
			continue
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, "->"))
		line, _ = trimSdkCurrentVersionMarker(line)

		if sdkName != "" {
			lowerLine = strings.ToLower(line)
			lowerSdkName := strings.ToLower(sdkName)
			switch {
			case strings.HasPrefix(lowerLine, lowerSdkName+"@"):
				line = strings.TrimSpace(line[len(sdkName)+1:])
			case strings.HasPrefix(lowerLine, lowerSdkName+" ->"):
				_, versionPart, _ := strings.Cut(line, "->")
				line = strings.TrimSpace(versionPart)
			case strings.HasPrefix(lowerLine, lowerSdkName+":"):
				line = strings.TrimSpace(line[len(sdkName)+1:])
			case strings.HasPrefix(lowerLine, lowerSdkName+" "):
				line = strings.TrimSpace(line[len(sdkName):])
			}
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, "@"))
		if fields := strings.Fields(line); len(fields) > 0 {
			line = fields[0]
		}
		line = strings.Trim(line, "\"'")
		if version := NormalizeSdkVersion(line); version != "" {
			return version
		}
	}
	return ""
}
