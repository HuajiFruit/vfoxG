//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// psEscape escapes a string for safe interpolation into a PowerShell single-quoted string.
func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func runElevatedScriptHelper(scriptContent string, doneFile string) error {
	tmpScript, tempErr := os.CreateTemp("", "vfox_elevated_*.ps1")
	if tempErr != nil {
		return fmt.Errorf("failed to create temp script: %v", tempErr)
	}
	tmpScriptPath := tmpScript.Name()
	if _, err := tmpScript.Write([]byte(scriptContent)); err != nil {
		tmpScript.Close()
		os.Remove(tmpScriptPath)
		return fmt.Errorf("failed to write temp script: %v", err)
	}
	if err := tmpScript.Close(); err != nil {
		os.Remove(tmpScriptPath)
		return fmt.Errorf("failed to close temp script: %v", err)
	}
	defer os.Remove(tmpScriptPath)

	psCmd := fmt.Sprintf(`$ErrorActionPreference = 'Stop'; try { Start-Process powershell -WindowStyle Hidden -Verb RunAs -ArgumentList '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', '%s' -Wait } catch { exit 1 }`, psEscape(tmpScriptPath))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	hideWindow(cmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("elevation failed or user cancelled: %v", err)
	}
	if _, err := os.Stat(doneFile); err != nil {
		return fmt.Errorf("script did not complete successfully")
	}
	return nil
}

func tempDoneFile(prefix string, name string) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	safeName := replacer.Replace(name)
	if safeName == "" {
		safeName = "sdk"
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s.done", prefix, safeName))
}
