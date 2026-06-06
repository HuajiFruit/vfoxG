//go:build !windows

package app

import (
	"os/exec"
)

func hideWindow(cmd *exec.Cmd) {
}

func getVfoxExeName() string {
	return "vfox"
}

func findExecutable(exe string, cleanEnv []string) string {
	candidates := findExecutableCandidates(exe, cleanEnv)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}
