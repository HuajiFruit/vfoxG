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

func (a *App) AddVfoxToPath() error {
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

func (a *App) RemoveVfoxFromPath() error {
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

func windowsSafeShimName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if name == "" {
		return "sdk"
	}
	return name
}

func windowsUniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func windowsSDKShimAliases(pluginName string) []string {
	aliases := []string{pluginName}
	for _, def := range systemSDKDefs {
		if def.Name == pluginName {
			aliases = append(aliases, def.Exe)
			break
		}
	}

	switch strings.ToLower(pluginName) {
	case "python":
		aliases = append(aliases, "python3", "pip", "pip3")
	case "nodejs":
		aliases = append(aliases, "npm", "npx", "corepack")
	case "java":
		aliases = append(aliases, "javac", "jar", "jshell", "jlink", "jpackage", "keytool")
	case "golang":
		aliases = append(aliases, "gofmt")
	case "rust":
		aliases = append(aliases, "cargo", "rustdoc", "rustup")
	case "ruby":
		aliases = append(aliases, "gem", "bundle", "bundler", "irb", "rake")
	case "php":
		aliases = append(aliases, "composer")
	case "perl":
		aliases = append(aliases, "cpan")
	}
	return windowsUniqueStrings(aliases)
}

func (a *App) windowsPathShimDir() string {
	return a.getVfoxHomePath("path-shims")
}

func windowsShimScript(pluginName string, alias string, sdkPath string) string {
	alias = windowsSafeShimName(alias)
	return fmt.Sprintf(`@echo off
setlocal
set "SDK_ROOT=%[1]s"
set "ALIAS_NAME=%[2]s"
if exist "%%SDK_ROOT%%\%[2]s.exe" ("%%SDK_ROOT%%\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.exe" ("%%SDK_ROOT%%\bin\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.exe" ("%%SDK_ROOT%%\Scripts\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\sbin\%[2]s.exe" ("%%SDK_ROOT%%\sbin\%[2]s.exe" %%* & exit /b)
if exist "%%SDK_ROOT%%\%[2]s.cmd" (call "%%SDK_ROOT%%\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.cmd" (call "%%SDK_ROOT%%\bin\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.cmd" (call "%%SDK_ROOT%%\Scripts\%[2]s.cmd" %%* & exit /b)
if exist "%%SDK_ROOT%%\%[2]s.bat" (call "%%SDK_ROOT%%\%[2]s.bat" %%* & exit /b)
if exist "%%SDK_ROOT%%\bin\%[2]s.bat" (call "%%SDK_ROOT%%\bin\%[2]s.bat" %%* & exit /b)
if exist "%%SDK_ROOT%%\Scripts\%[2]s.bat" (call "%%SDK_ROOT%%\Scripts\%[2]s.bat" %%* & exit /b)
for /f "delims=" %%%%I in ('where "%%ALIAS_NAME%%" 2^>nul') do (
  if /I not "%%%%~fI"=="%%~f0" (
    if /I not "%%%%~dpI"=="%%~dp0" (
      if /I "%%%%~xI"==".cmd" (call "%%%%~fI" %%* & exit /b)
      if /I "%%%%~xI"==".bat" (call "%%%%~fI" %%* & exit /b)
      "%%%%~fI" %%*
      exit /b
    )
  )
)
echo vfoxG: %[2]s for %[3]s is not available under %%SDK_ROOT%%, and no fallback %[2]s was found on PATH. 1>&2
exit /b 9009
`, sdkPath, alias, pluginName)
}

func (a *App) writeWindowsSDKShims(pluginName string) ([]string, error) {
	if strings.TrimSpace(pluginName) == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}
	shimDir := a.windowsPathShimDir()
	if shimDir == "" {
		return nil, fmt.Errorf("unable to resolve shim directory")
	}
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return nil, err
	}
	aliases := windowsSDKShimAliases(pluginName)
	sdkPath := a.getVfoxHomePath("sdks", pluginName)
	for _, alias := range aliases {
		shimName := windowsSafeShimName(alias) + ".cmd"
		shimPath := filepath.Join(shimDir, shimName)
		if err := os.WriteFile(shimPath, []byte(windowsShimScript(pluginName, alias, sdkPath)), 0644); err != nil {
			return nil, err
		}
	}
	return aliases, nil
}

func (a *App) removeWindowsSDKShims(pluginName string, aliases []string) error {
	shimDir := a.windowsPathShimDir()
	if shimDir == "" {
		return nil
	}
	if len(aliases) == 0 {
		aliases = windowsSDKShimAliases(pluginName)
	}
	for _, alias := range aliases {
		_ = os.Remove(filepath.Join(shimDir, windowsSafeShimName(alias)+".cmd"))
	}
	return nil
}

func (a *App) HijackSystemPath(name string, exePath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}
	if strings.TrimSpace(exePath) != "" {
		if err := validateSDKExecutablePath(exePath); err != nil {
			return err
		}
		sdkRoot := a.getSdkRoot(exePath)
		sdkLinkPath := a.getVfoxHomePath("sdks", name)
		if err := a.ensureJunction(sdkLinkPath, sdkRoot); err != nil {
			return err
		}
	}
	aliases, err := a.writeWindowsSDKShims(name)
	if err != nil {
		return err
	}

	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	shimDir := a.windowsPathShimDir()
	aliasesJSON, _ := json.Marshal(aliases)
	doneFile := tempDoneFile("vfox_hijack", name)
	os.Remove(doneFile)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$hijackFile = '%s'
$name = '%s'
$shimDir = '%s'
$aliasesJson = '%s'

function Normalize-Path($p) {
    if (-not $p) { return '' }
    return $p.Trim().TrimEnd('\')
}

function Add-MachinePathEntry($target) {
    $current = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    if (-not $current) {
        $parts = @()
    } else {
        $parts = @($current -split ';' | Where-Object { $_.Trim() -ne '' })
    }
    $normalizedTarget = Normalize-Path $target
    $cleaned = @()
    foreach ($p in $parts) {
        if ((Normalize-Path $p).ToLower() -ne $normalizedTarget.ToLower()) {
            $cleaned += $p.Trim()
        }
    }
    $newPath = @($target) + $cleaned
    [Environment]::SetEnvironmentVariable('Path', ($newPath -join ';'), 'Machine')
}

Add-MachinePathEntry $shimDir
$aliases = @()
if ($aliasesJson) {
    $aliases = @($aliasesJson | ConvertFrom-Json)
}

$allData = New-Object PSObject
if (Test-Path $hijackFile) {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$data = @{
    Version = 2
    ShimDir = $shimDir
    Aliases = @($aliases)
}
$allData | Add-Member -MemberType NoteProperty -Name $name -Value $data -Force
$allData | ConvertTo-Json -Depth 10 | Set-Content $hijackFile

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(hijackFile), psEscape(name), psEscape(shimDir), psEscape(string(aliasesJSON)), psEscape(doneFile))

	if err := runElevatedScriptHelper(script, doneFile); err != nil {
		_ = a.removeWindowsSDKShims(name, aliases)
		return err
	}
	return nil
}

func (a *App) RestoreSystemPath(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	aliases := windowsSDKShimAliases(name)
	if data, err := os.ReadFile(hijackFile); err == nil {
		var parsed map[string]struct {
			Aliases []string `json:"Aliases"`
		}
		if json.Unmarshal(data, &parsed) == nil && len(parsed[name].Aliases) > 0 {
			aliases = parsed[name].Aliases
		}
	}
	vfoxSdksDir := a.getVfoxHomePath("sdks")
	shimDir := a.windowsPathShimDir()
	doneFile := tempDoneFile("vfox_restore", name)
	os.Remove(doneFile)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$hijackFile = '%s'
$name = '%s'
$vfoxSdksDir = '%s'
$shimDir = '%s'

if (-not (Test-Path $hijackFile)) {
    $allData = New-Object PSObject
} else {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$data = $null
if ($null -ne $allData -and $null -ne $allData.PSObject.Properties[$name]) {
    $data = $allData.PSObject.Properties[$name].Value
}

function Normalize-Path($p) {
    if (-not $p) { return '' }
    return $p.Trim().TrimEnd('\')
}

function Restore-Paths($paths, $scope) {
    if (-not $paths -or $paths.Count -eq 0) { return }
    $current = [Environment]::GetEnvironmentVariable('Path', $scope)
    if (-not $current) { $current = '' }
    $parts = @($current -split ';' | Where-Object { $_.Trim() -ne '' })
    $newParts = @($parts)
    foreach ($p in $paths) {
        $pTrim = $p.Trim()
        if ($pTrim -eq '') { continue }
        $exists = $false
        foreach ($existing in $newParts) {
            if ((Normalize-Path $existing).ToLower() -eq (Normalize-Path $pTrim).ToLower()) {
                $exists = $true
                break
            }
        }
        if (-not $exists) {
            $newParts = @($pTrim) + $newParts
        }
    }
    [Environment]::SetEnvironmentVariable('Path', ($newParts -join ';'), $scope)
}

function Remove-MachinePathEntries($paths) {
    $current = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    if (-not $current) { return }
    $removeSet = @{}
    foreach ($p in $paths) {
        $normalized = (Normalize-Path $p).ToLower()
        if ($normalized -ne '') {
            $removeSet[$normalized] = $true
        }
    }
    $parts = @($current -split ';' | Where-Object { $_.Trim() -ne '' })
    $cleaned = @()
    foreach ($p in $parts) {
        $normalized = (Normalize-Path $p).ToLower()
        if (-not $removeSet.ContainsKey($normalized)) {
            $cleaned += $p.Trim()
        }
    }
    [Environment]::SetEnvironmentVariable('Path', ($cleaned -join ';'), 'Machine')
}

function Managed-PluginCount($obj) {
    if ($null -eq $obj) { return 0 }
    return @($obj.PSObject.Properties).Count
}

$legacyUserPaths = @()
$legacyMachinePaths = @()
if ($data) {
    if ($null -ne $data.PSObject.Properties['UserPaths']) {
        $legacyUserPaths = @($data.UserPaths)
    }
    if ($null -ne $data.PSObject.Properties['MachinePaths']) {
        $legacyMachinePaths = @($data.MachinePaths)
    }
}

if ($null -ne $allData.PSObject.Properties[$name]) {
    $allData.PSObject.Properties.Remove($name)
}

$vfoxPath = Join-Path $vfoxSdksDir $name
$legacyManagedPaths = @($vfoxPath, (Join-Path $vfoxPath 'Scripts'), (Join-Path $vfoxPath 'bin'))
if (Managed-PluginCount $allData -eq 0) {
    $legacyManagedPaths += $shimDir
}
Remove-MachinePathEntries $legacyManagedPaths

Restore-Paths $legacyUserPaths 'User'
Restore-Paths $legacyMachinePaths 'Machine'

if (Managed-PluginCount $allData -eq 0) {
    if (Test-Path $hijackFile) {
        Remove-Item $hijackFile -Force
    }
} else {
    $allData | ConvertTo-Json -Depth 10 | Set-Content $hijackFile
}

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(hijackFile), psEscape(name), psEscape(vfoxSdksDir), psEscape(shimDir), psEscape(doneFile))

	if err := runElevatedScriptHelper(script, doneFile); err != nil {
		return err
	}

	return a.removeWindowsSDKShims(name, aliases)
}

func (a *App) detachPluginPathOverrideFiles(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	aliases := windowsSDKShimAliases(name)
	if data, err := os.ReadFile(hijackFile); err == nil {
		var parsed map[string]struct {
			Aliases []string `json:"Aliases"`
		}
		if json.Unmarshal(data, &parsed) == nil && len(parsed[name].Aliases) > 0 {
			aliases = parsed[name].Aliases
		}
	}

	if err := a.removeWindowsSDKShims(name, aliases); err != nil {
		return err
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if sdkLinkPath == "" {
		return fmt.Errorf("unable to resolve vfox home directory")
	}
	a.removeJunctionIfExists(sdkLinkPath)
	return nil
}

func (a *App) HijackPluginSystemPath(pluginName string) error {
	m := a.GetNonVfoxSdksMap()
	if list, ok := m[pluginName]; ok && len(list) > 0 {
		return a.HijackSystemPath(pluginName, list[0].Path)
	}
	return a.HijackSystemPath(pluginName, "")
}

func (a *App) RestorePluginSystemPath(pluginName string) error {
	return a.RestoreSystemPath(pluginName)
}

func (a *App) CheckPluginWin11CompatMode(pluginName string) bool {
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if hijackFile == "" {
		return false
	}
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
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if hijackFile == "" {
		return false
	}
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
	candidates := findExecutableCandidates(exe, cleanEnv)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

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
	var candidates []string
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
