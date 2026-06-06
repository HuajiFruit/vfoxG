package parser

import "strings"

func SearchVersions(commandOutput string) []string {
	lines := strings.Split(commandOutput, "\n")
	versions := make([]string, 0)
	seenVersions := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || isSearchOutputHeaderLine(line) {
			continue
		}

		version := firstSearchVersionToken(strings.Fields(line))
		if version != "" && version != "Error:" && version != "Available" && !seenVersions[version] {
			versions = append(versions, version)
			seenVersions[version] = true
		}
	}

	return versions
}

func isSearchOutputHeaderLine(line string) bool {
	return strings.HasPrefix(line, "Available") ||
		strings.HasPrefix(line, "Search") ||
		strings.HasPrefix(line, "Please") ||
		strings.HasPrefix(line, "Use") ||
		strings.HasPrefix(line, "Name") ||
		strings.HasPrefix(line, "---")
}

func firstSearchVersionToken(tokens []string) string {
	for _, token := range tokens {
		version := strings.TrimSpace(strings.Trim(token, ",;"))
		if version == "" || version == "-" || version == "*" || version == "•" {
			continue
		}
		if tokenHasDigit(version) {
			return version
		}
	}
	return ""
}

func tokenHasDigit(token string) bool {
	for _, char := range token {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}
