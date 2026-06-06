//go:build !windows

package app

import (
	"os"
	"path/filepath"
	"strings"
)

func findExecutableCandidates(exe string, cleanEnv []string) []string {
	if strings.ContainsRune(exe, filepath.Separator) {
		if isExecutableFile(exe) {
			return []string{exe}
		}
		return nil
	}

	pathValue := os.Getenv("PATH")
	for _, envValue := range cleanEnv {
		if strings.HasPrefix(envValue, "PATH=") {
			pathValue = strings.TrimPrefix(envValue, "PATH=")
			break
		}
	}

	candidates := make([]string, 0)
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, exe)
		if isExecutableFile(candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}
