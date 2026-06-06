//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a *App) checkVfoxInPath() (bool, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetEnvironmentVariable('Path', 'User')")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	vfoxCoreDir := a.getCoreDir()
	for _, entry := range strings.Split(strings.TrimSpace(string(out)), ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.EqualFold(filepath.Clean(entry), filepath.Clean(vfoxCoreDir)) {
			return true, nil
		}
	}
	return false, nil
}

func (a *App) addVfoxToPath() error {
	vfoxCoreDir := a.getCoreDir()
	escDir := psEscape(vfoxCoreDir)
	script := fmt.Sprintf(`
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not $currentPath) {
    $paths = @()
} else {
    $paths = @($currentPath -split ';' | Where-Object { $_.Trim() -ne '' })
}
$target = '%s'
$normalizedTarget = $target.Trim().TrimEnd('\')
$exists = $false
foreach ($p in $paths) {
    if ([string]::Equals($p.Trim().TrimEnd('\'), $normalizedTarget, [System.StringComparison]::OrdinalIgnoreCase)) {
        $exists = $true
        break
    }
}
if (-not $exists) {
    $paths += $target
    [Environment]::SetEnvironmentVariable('Path', ($paths -join ';'), 'User')
}
	`, escDir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	hideWindow(cmd)
	return cmd.Run()
}

func (a *App) removeVfoxFromPath() error {
	vfoxCoreDir := a.getCoreDir()
	escDir := psEscape(vfoxCoreDir)
	script := fmt.Sprintf(`
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not $currentPath) {
    return
}
$target = '%s'
$normalizedTarget = $target.Trim().TrimEnd('\')
$paths = @($currentPath -split ';' | Where-Object {
    $entry = $_.Trim()
    $entry -ne '' -and -not [string]::Equals($entry.TrimEnd('\'), $normalizedTarget, [System.StringComparison]::OrdinalIgnoreCase)
})
[Environment]::SetEnvironmentVariable('Path', ($paths -join ';'), 'User')
	`, escDir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	hideWindow(cmd)
	return cmd.Run()
}
