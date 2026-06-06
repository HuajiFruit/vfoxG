//go:build windows

package app

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

func getVfoxExeName() string {
	return "vfox.exe"
}

func findExecutable(exe string, cleanEnv []string) string {
	candidates := findExecutableCandidates(exe, cleanEnv)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}
