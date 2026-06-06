package app

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func (a *App) runVfoxCommandWithLock(args ...string) (string, error) {
	if !isExclusiveVfoxCommand(args) {
		return a.runVfoxCommand(args...)
	}

	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()
	return a.runVfoxCommand(args...)
}

func isExclusiveVfoxCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "add", "install", "remove", "uninstall", "use", "unuse":
		return true
	default:
		return false
	}
}

func (a *App) runVfoxCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	vfoxExe, err := a.getVfoxExecutable()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, vfoxExe, args...)

	hideWindow(cmd)
	// Pretend to be a shell process so vfox does not relaunch the GUI parent.
	cmd.Env = a.getCleanedEnvForVfox()

	out, err := cmd.CombinedOutput()

	cleanOut := ansiRegex.ReplaceAllString(string(out), "")

	if ctx.Err() == context.DeadlineExceeded {
		return cleanOut, fmt.Errorf("vfox %v timed out after 15s", args)
	}

	if err != nil {
		return cleanOut, fmt.Errorf("command failed: %w, output: %s", err, cleanOut)
	}

	return cleanOut, nil
}
