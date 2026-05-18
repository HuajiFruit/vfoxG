//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func getVfoxExeName() string {
	return "vfox.exe"
}

// psEscape escapes a string for safe interpolation into a PowerShell single-quoted string.
func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func (a *App) ensureJunction(linkPath string, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0 {
			existing, readErr := os.Readlink(linkPath)
			if readErr == nil && strings.EqualFold(filepath.Clean(existing), filepath.Clean(target)) {
				return nil
			}
		} else if fi.IsDir() {
			if strings.EqualFold(filepath.Clean(linkPath), filepath.Clean(target)) {
				return nil
			}
		}
	}
	_ = os.RemoveAll(linkPath)
	_ = os.MkdirAll(filepath.Dir(linkPath), 0755)

	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, target)
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mklink /J failed: %v", err)
	}
	return nil
}

func (a *App) CheckVfoxInPath() (bool, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetEnvironmentVariable('Path', 'User')")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	pathStr := string(out)
	vfoxCoreDir := a.getCoreDir()
	if strings.Contains(pathStr, vfoxCoreDir) {
		return true, nil
	}
	return false, nil
}

func (a *App) AddVfoxToPath() error {
	vfoxCoreDir := a.getCoreDir()
	escDir := psEscape(vfoxCoreDir)
	script := fmt.Sprintf(`
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($currentPath -notlike '*%s*') {
    $newPath = $currentPath
    if (-not $newPath.EndsWith(';')) {
        $newPath += ';'
    }
    $newPath += '%s'
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
}
	`, escDir, escDir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	hideWindow(cmd)
	return cmd.Run()
}

func (a *App) RemoveVfoxFromPath() error {
	vfoxCoreDir := a.getCoreDir()
	escDir := psEscape(vfoxCoreDir)
	script := fmt.Sprintf(`
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($currentPath -like '*%s*') {
    $paths = $currentPath -split ';'
    $newPaths = $paths | Where-Object { $_ -ne '%s' -and $_ -ne '' }
    $newPath = $newPaths -join ';'
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
}
	`, escDir, escDir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	hideWindow(cmd)
	return cmd.Run()
}

func runElevatedScriptHelper(scriptContent string, doneFile string) error {
	tmpScript, tempErr := os.CreateTemp("", "vfox_elevated_*.ps1")
	if tempErr != nil {
		return fmt.Errorf("failed to create temp script: %v", tempErr)
	}
	tmpScriptPath := tmpScript.Name()
	tmpScript.Write([]byte(scriptContent))
	tmpScript.Close()
	defer os.Remove(tmpScriptPath)

	psCmd := fmt.Sprintf(`$ErrorActionPreference = 'Stop'; try { Start-Process powershell -WindowStyle Hidden -Verb RunAs -ArgumentList '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', '%s' -Wait } catch { exit 1 }`, tmpScriptPath)
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

func (a *App) HijackSystemPath(name string, exePath string) error {
	sdkDir := a.getSdkRoot(exePath)
	vfoxHome := a.getVfoxHome()
	hijackFile := filepath.Join(vfoxHome, "hijacked_paths.json")
	doneFile := filepath.Join(os.TempDir(), fmt.Sprintf("vfox_hijack_%s.done", name))
	os.Remove(doneFile)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$sdkDir = '%s'
$hijackFile = '%s'
$name = '%s'
$vfoxSdksDir = '%s'

function Clean-Path($targetPath, $scope) {
    $current = [Environment]::GetEnvironmentVariable('Path', $scope)
    if (-not $current) { return @() }
    
    $parts = $current -split ';'
    $cleaned = @()
    $removed = @()
    
    foreach ($p in $parts) {
        $pTrim = $p.Trim()
        if ($pTrim -eq '') { continue }
        if ($pTrim.ToLower().StartsWith($sdkDir.ToLower())) {
            $removed += $pTrim
        } else {
            $cleaned += $pTrim
        }
    }
    
    if ($scope -eq 'Machine') {
        $vfoxPath = Join-Path $vfoxSdksDir $name
        $vfoxScripts = Join-Path $vfoxPath 'Scripts'
        $vfoxBin = Join-Path $vfoxPath 'bin'
        
        $insert = @($vfoxPath, $vfoxScripts, $vfoxBin)
        $finalCleaned = @()
        foreach ($p in $insert) {
            $finalCleaned += $p
        }
        foreach ($p in $cleaned) {
            $pLower = $p.ToLower()
            if ($pLower -ne $vfoxPath.ToLower() -and $pLower -ne $vfoxScripts.ToLower() -and $pLower -ne $vfoxBin.ToLower()) {
                $finalCleaned += $p
            }
        }
        $cleaned = $finalCleaned
    }
    
    if ($removed.Length -gt 0 -or $scope -eq 'Machine') {
        $newPath = $cleaned -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newPath, $scope)
    }
    return $removed
}

$removedUser = Clean-Path $sdkDir 'User'
$removedMachine = Clean-Path $sdkDir 'Machine'

$data = @{
    UserPaths = @($removedUser)
    MachinePaths = @($removedMachine)
}

$json = $data | ConvertTo-Json -Depth 10

$allData = New-Object PSObject
if (Test-Path $hijackFile) {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$allData | Add-Member -MemberType NoteProperty -Name $name -Value $data -Force
$allData | ConvertTo-Json -Depth 10 | Set-Content $hijackFile

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(sdkDir), psEscape(hijackFile), psEscape(name), psEscape(filepath.Join(vfoxHome, "sdks")), psEscape(doneFile))

	return runElevatedScriptHelper(script, doneFile)
}

func (a *App) RestoreSystemPath(name string) error {
	vfoxHome := a.getVfoxHome()
	hijackFile := filepath.Join(vfoxHome, "hijacked_paths.json")
	doneFile := filepath.Join(os.TempDir(), fmt.Sprintf("vfox_restore_%s.done", name))
	os.Remove(doneFile)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$hijackFile = '%s'
$name = '%s'
$vfoxSdksDir = '%s'

if (-not (Test-Path $hijackFile)) {
    $allData = New-Object PSObject
} else {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$data = $null
if ($null -ne $allData -and $null -ne $allData.PSObject.Properties[$name]) {
    $data = $allData.PSObject.Properties[$name].Value
}

function Restore-Path($paths, $scope) {
    if ((-not $paths -or $paths.Count -eq 0) -and $scope -ne 'Machine') { return }
    $current = [Environment]::GetEnvironmentVariable('Path', $scope)
    if (-not $current) { $current = '' }
    $parts = $current -split ';'
    $cleaned = @()
    $vfoxPath = Join-Path $vfoxSdksDir $name
    $vfoxScripts = Join-Path $vfoxPath 'Scripts'
    $vfoxBin = Join-Path $vfoxPath 'bin'

    foreach ($p in $parts) {
        $pTrim = $p.Trim()
        if ($pTrim -eq '') { continue }
        if ($scope -eq 'Machine') {
            $pLower = $pTrim.ToLower()
            if ($pLower -eq $vfoxPath.ToLower() -or $pLower -eq $vfoxScripts.ToLower() -or $pLower -eq $vfoxBin.ToLower()) {
                continue
            }
        }
        $cleaned += $pTrim
    }

    $newPath = $cleaned -join ';'
    if ($paths) {
        foreach ($p in $paths) {
            if (-not ($newPath.ToLower().Contains($p.ToLower()))) {
                if ($newPath.Length -gt 0 -and -not $newPath.EndsWith(';')) {
                    $newPath = $p + ';' + $newPath
                } else {
                    $newPath = $p + ';' + $newPath
                }
            }
        }
    }
    [Environment]::SetEnvironmentVariable('Path', $newPath, $scope)
}

if ($data) {
    Restore-Path $data.UserPaths 'User'
    Restore-Path $data.MachinePaths 'Machine'
} else {
    Restore-Path $null 'Machine'
}

if ($null -ne $allData.PSObject.Properties[$name]) {
    $allData.PSObject.Properties.Remove($name)
    $allData | ConvertTo-Json -Depth 10 | Set-Content $hijackFile
}

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(hijackFile), psEscape(name), psEscape(filepath.Join(vfoxHome, "sdks")), psEscape(doneFile))

	return runElevatedScriptHelper(script, doneFile)
}

func (a *App) HijackPluginSystemPath(pluginName string) error {
	m := a.GetNonVfoxSdksMap()
	if list, ok := m[pluginName]; ok && len(list) > 0 {
		return a.HijackSystemPath(pluginName, list[0].Path)
	}
	return a.HijackSystemPath(pluginName, "C:\\VFOX_DUMMY_NEVER_MATCH\\fake.exe")
}

func (a *App) RestorePluginSystemPath(pluginName string) error {
	return a.RestoreSystemPath(pluginName)
}

func (a *App) CheckPluginWin11CompatMode(pluginName string) bool {
	vfoxHome := a.getVfoxHome()
	hijackFile := filepath.Join(vfoxHome, "hijacked_paths.json")
	if _, err := os.Stat(hijackFile); err == nil {
		data, err := os.ReadFile(hijackFile)
		if err == nil {
			var parsed map[string]interface{}
			if json.Unmarshal(data, &parsed) == nil {
				_, exists := parsed[pluginName]
				return exists
			}
		}
	}
	return false
}

func (a *App) CheckWin11CompatMode() bool {
	vfoxHome := a.getVfoxHome()
	hijackFile := filepath.Join(vfoxHome, "hijacked_paths.json")
	if _, err := os.Stat(hijackFile); err == nil {
		data, err := os.ReadFile(hijackFile)
		if err == nil {
			var parsed map[string]interface{}
			if json.Unmarshal(data, &parsed) == nil {
				return len(parsed) > 0
			}
		}
	}
	return false
}

func findExecutable(exe string, cleanEnv []string) string {
	lookCmd := exec.Command("cmd", "/c", "where", exe)
	hideWindow(lookCmd)
	lookCmd.Env = cleanEnv
	whereOut, err := lookCmd.Output()
	if err != nil {
		return ""
	}
	exePath := strings.TrimSpace(strings.Split(string(whereOut), "\n")[0])
	exePath = strings.TrimRight(exePath, "\r")
	return exePath
}
