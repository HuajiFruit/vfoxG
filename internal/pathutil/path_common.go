package pathutil

import (
	"path/filepath"
	"strings"
)

func CleanPathValue(pathValue string, excludedRoots []string) string {
	parts := filepath.SplitList(pathValue)
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		pathPart := strings.TrimSpace(part)
		if pathPart == "" {
			continue
		}
		excluded := false
		for _, root := range excludedRoots {
			if IsPathWithin(pathPart, root) {
				excluded = true
				break
			}
		}
		if !excluded {
			cleanParts = append(cleanParts, pathPart)
		}
	}
	return strings.Join(cleanParts, string(filepath.ListSeparator))
}
