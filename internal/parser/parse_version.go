package parser

import "strings"

func NormalizeSdkVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}

func SameSdkVersion(leftVersion string, rightVersion string) bool {
	return strings.EqualFold(NormalizeSdkVersion(leftVersion), NormalizeSdkVersion(rightVersion))
}

func trimSdkCurrentVersionMarker(line string) (string, bool) {
	for _, marker := range []string{"<-- current", "<— current", "<- current"} {
		if strings.HasSuffix(line, marker) {
			return strings.TrimSpace(strings.TrimSuffix(line, marker)), true
		}
	}
	return line, false
}
