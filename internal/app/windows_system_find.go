//go:build windows

package app

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func findExecutableCandidates(exe string, cleanEnv []string) []string {
	lookCmd := exec.Command("cmd", "/c", "where", exe)
	hideWindow(lookCmd)
	lookCmd.Env = cleanEnv
	whereOut, err := lookCmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(whereOut), "\n")
	seen := make(map[string]bool)
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		exePath := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if exePath == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(exePath))
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, exePath)
	}
	return candidates
}
