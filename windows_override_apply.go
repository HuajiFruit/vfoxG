//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type windowsPathApplyContext struct {
	Name        string
	HijackFile  string
	ShimDir     string
	AliasesJSON string
	DoneFile    string
}

func (a *App) refreshActiveSdkPathOverride(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || !a.checkPluginPathOverride(name) {
		return nil
	}
	aliases, err := a.writeWindowsSDKShims(name)
	if err != nil {
		return err
	}
	return a.writeWindowsPathOverrideEntry(name, aliases)
}

func (a *App) refreshPathOverridesAfterVfoxHomeChange(oldHome string) error {
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if strings.TrimSpace(hijackFile) == "" {
		return nil
	}
	entries, err := readWindowsPathOverrideEntries(hijackFile)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	shimDir := a.windowsPathShimDir()
	if strings.TrimSpace(shimDir) == "" {
		return fmt.Errorf("unable to resolve shim directory")
	}

	removeRoots := windowsMigratedPathOverrideRoots(oldHome, shimDir, entries)
	refreshed := make(map[string]windowsPathOverrideEntry, len(entries))
	for name, entry := range entries {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		aliases, err := a.writeWindowsSDKShims(name)
		if err != nil {
			return err
		}
		entry.Version = 2
		entry.ShimDir = shimDir
		entry.Aliases = aliases
		refreshed[name] = entry
	}
	if len(refreshed) == 0 {
		return nil
	}
	if err := a.writeJSONFile(hijackFile, refreshed); err != nil {
		return err
	}
	return updateWindowsMigratedPathOverride(removeRoots, shimDir)
}

func windowsMigratedPathOverrideRoots(oldHome string, shimDir string, entries map[string]windowsPathOverrideEntry) []string {
	var roots []string
	addRoot := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		for _, existing := range roots {
			if strings.EqualFold(filepath.Clean(existing), path) {
				return
			}
		}
		roots = append(roots, path)
	}

	addRoot(shimDir)
	if strings.TrimSpace(oldHome) != "" {
		addRoot(filepath.Join(oldHome, "path-shims"))
		for name := range entries {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			sdkPath := filepath.Join(oldHome, "sdks", name)
			addRoot(sdkPath)
			addRoot(filepath.Join(sdkPath, "Scripts"))
			addRoot(filepath.Join(sdkPath, "bin"))
			addRoot(filepath.Join(sdkPath, "sbin"))
		}
	}
	for _, entry := range entries {
		addRoot(entry.ShimDir)
	}
	return roots
}

func updateWindowsMigratedPathOverride(removeRoots []string, shimDir string) error {
	rootsJSON, _ := json.Marshal(removeRoots)
	doneFile := tempDoneFile("vfox_migrate_path", "sdk")
	os.Remove(doneFile)

	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$removeRootsJson = '%s'
$shimDir = '%s'

function Normalize-Path($p) {
    if (-not $p) { return '' }
    return $p.Trim().TrimEnd('\')
}

$removeRoots = @()
if ($removeRootsJson) {
    $removeRoots = @($removeRootsJson | ConvertFrom-Json)
}
$removeSet = @{}
foreach ($p in $removeRoots) {
    $normalized = (Normalize-Path $p).ToLower()
    if ($normalized -ne '') {
        $removeSet[$normalized] = $true
    }
}

$current = [Environment]::GetEnvironmentVariable('Path', 'Machine')
if (-not $current) {
    $parts = @()
} else {
    $parts = @($current -split ';' | Where-Object { $_.Trim() -ne '' })
}

$cleaned = @()
foreach ($p in $parts) {
    $normalized = (Normalize-Path $p).ToLower()
    if ($normalized -ne '' -and -not $removeSet.ContainsKey($normalized)) {
        $cleaned += $p.Trim()
    }
}

$newParts = @($shimDir) + $cleaned
[Environment]::SetEnvironmentVariable('Path', ($newParts -join ';'), 'Machine')

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(string(rootsJSON)), psEscape(shimDir), psEscape(doneFile))

	return runElevatedScriptHelper(script, doneFile)
}

func (a *App) hijackSystemPath(name string, exePath string) error {
	name, err := a.prepareWindowsPathApplyTarget(name, exePath)
	if err != nil {
		return err
	}
	aliases, err := a.writeWindowsSDKShims(name)
	if err != nil {
		return err
	}

	applyContext := a.newWindowsPathApplyContext(name, aliases)
	if err := runElevatedScriptHelper(buildWindowsPathApplyScript(applyContext), applyContext.DoneFile); err != nil {
		_ = a.removeWindowsSDKShims(name, aliases)
		return err
	}
	return nil
}

func (a *App) prepareWindowsPathApplyTarget(name string, exePath string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return "", err
	}
	if strings.TrimSpace(exePath) == "" {
		return name, nil
	}
	if err := validateSDKExecutablePath(exePath); err != nil {
		return "", err
	}
	sdkRoot := a.getSdkRoot(exePath)
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if err := a.ensureJunction(sdkLinkPath, sdkRoot); err != nil {
		return "", err
	}
	return name, nil
}

func (a *App) newWindowsPathApplyContext(name string, aliases []string) windowsPathApplyContext {
	aliasesJSON, _ := json.Marshal(aliases)
	doneFile := tempDoneFile("vfox_hijack", name)
	os.Remove(doneFile)

	return windowsPathApplyContext{
		Name:        name,
		HijackFile:  a.getVfoxHomePath("hijacked_paths.json"),
		ShimDir:     a.windowsPathShimDir(),
		AliasesJSON: string(aliasesJSON),
		DoneFile:    doneFile,
	}
}

func buildWindowsPathApplyScript(context windowsPathApplyContext) string {
	return fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$hijackFile = '%s'
$name = '%s'
$shimDir = '%s'
$aliasesJson = '%s'

%s

%s

New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(context.HijackFile), psEscape(context.Name), psEscape(context.ShimDir), psEscape(context.AliasesJSON), windowsPathApplyFunctionsScript(), windowsPathApplyMainScript(), psEscape(context.DoneFile))
}

func windowsPathApplyFunctionsScript() string {
	return `
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
`
}

func windowsPathApplyMainScript() string {
	return `
Add-MachinePathEntry $shimDir
if ($aliasesJson) {
    $aliases = $aliasesJson | ConvertFrom-Json
} else {
    $aliases = @()
}

$allData = New-Object PSObject
if (Test-Path $hijackFile) {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$data = @{
    Version = 2
    ShimDir = $shimDir
    Aliases = $aliases
}
$allData | Add-Member -MemberType NoteProperty -Name $name -Value $data -Force
$allData | ConvertTo-Json -Depth 10 | Set-Content $hijackFile

Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
$result = [UIntPtr]::Zero
[Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
`
}

func (a *App) hijackPluginSystemPath(pluginName string) error {
	m := a.getNonVfoxSdksMap()
	if list, ok := m[pluginName]; ok && len(list) > 0 {
		return a.hijackSystemPath(pluginName, list[0].Path)
	}
	return a.hijackSystemPath(pluginName, "")
}
