//go:build windows

package app

import (
	"fmt"
	"os"
	"strings"
)

type windowsPathRestoreContext struct {
	Name        string
	HijackFile  string
	VfoxSdksDir string
	ShimDir     string
	DoneFile    string
	Aliases     []string
}

func (a *App) restoreSystemPath(name string) error {
	restoreContext, err := a.prepareWindowsPathRestore(name)
	if err != nil {
		return err
	}

	if err := runElevatedScriptHelper(buildWindowsPathRestoreScript(restoreContext), restoreContext.DoneFile); err != nil {
		return err
	}

	return a.removeWindowsSDKShims(restoreContext.Name, restoreContext.Aliases)
}

func (a *App) prepareWindowsPathRestore(name string) (windowsPathRestoreContext, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return windowsPathRestoreContext{}, fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return windowsPathRestoreContext{}, err
	}

	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	aliases := windowsSDKShimAliases(name)
	if parsedAliases, ok := readWindowsPathOverrideAliases(hijackFile, name); ok {
		aliases = parsedAliases
	}
	doneFile := tempDoneFile("vfox_restore", name)
	os.Remove(doneFile)

	return windowsPathRestoreContext{
		Name:        name,
		HijackFile:  hijackFile,
		VfoxSdksDir: a.getVfoxHomePath("sdks"),
		ShimDir:     a.windowsPathShimDir(),
		DoneFile:    doneFile,
		Aliases:     aliases,
	}, nil
}

func buildWindowsPathRestoreScript(context windowsPathRestoreContext) string {
	return fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$hijackFile = '%s'
$name = '%s'
$vfoxSdksDir = '%s'
$shimDir = '%s'

%s

%s

New-Item -Path '%s' -ItemType File -Force | Out-Null
`, psEscape(context.HijackFile), psEscape(context.Name), psEscape(context.VfoxSdksDir), psEscape(context.ShimDir), windowsPathRestoreFunctionsScript(), windowsPathRestoreMainScript(), psEscape(context.DoneFile))
}

func windowsPathRestoreFunctionsScript() string {
	return strings.Join([]string{
		windowsPathRestoreMetadataScript(),
		windowsPathNormalizeScript(),
		windowsPathRestorePathsScript(),
		windowsPathRemoveMachineEntriesScript(),
		windowsPathManagedPluginCountScript(),
	}, "\n")
}

func windowsPathRestoreMetadataScript() string {
	return `
if (-not (Test-Path $hijackFile)) {
    $allData = New-Object PSObject
} else {
    $allData = Get-Content $hijackFile -Raw | ConvertFrom-Json
}
$data = $null
if ($null -ne $allData -and $null -ne $allData.PSObject.Properties[$name]) {
    $data = $allData.PSObject.Properties[$name].Value
}
`
}

func windowsPathNormalizeScript() string {
	return `
function Normalize-Path($p) {
    if (-not $p) { return '' }
    return $p.Trim().TrimEnd('\')
}
`
}

func windowsPathRestorePathsScript() string {
	return `
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
`
}

func windowsPathRemoveMachineEntriesScript() string {
	return `
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
`
}

func windowsPathManagedPluginCountScript() string {
	return `
function Managed-PluginCount($obj) {
    if ($null -eq $obj) { return 0 }
    return @($obj.PSObject.Properties).Count
}
`
}

func windowsPathRestoreMainScript() string {
	return `
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
`
}

func (a *App) detachPluginPathOverrideFiles(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	aliases := windowsSDKShimAliases(name)
	if parsedAliases, ok := readWindowsPathOverrideAliases(hijackFile, name); ok {
		aliases = parsedAliases
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

func (a *App) restorePluginSystemPath(pluginName string) error {
	return a.restoreSystemPath(pluginName)
}
