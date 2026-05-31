param(
    [string]$InstallDir,
    [string]$ProductName,
    [string]$ProductExecutable,
    [switch]$RemoveSdkData
)

$ErrorActionPreference = 'Continue'

function Normalize-PathValue($Path) {
    if (-not $Path) { return '' }
    try {
        return [System.IO.Path]::GetFullPath($Path).TrimEnd('\').ToLowerInvariant()
    } catch {
        return $Path.Trim().TrimEnd('\').ToLowerInvariant()
    }
}

function Test-PathUnderRoot($Path, $Root) {
    $normalizedPath = Normalize-PathValue $Path
    $normalizedRoot = Normalize-PathValue $Root
    if (-not $normalizedPath -or -not $normalizedRoot) { return $false }
    return $normalizedPath -eq $normalizedRoot -or $normalizedPath.StartsWith($normalizedRoot + '\')
}

function Get-PathParts($Scope) {
    $value = [Environment]::GetEnvironmentVariable('Path', $Scope)
    if (-not $value) { return @() }
    return @($value -split ';' | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne '' })
}

function Set-PathParts($Scope, $Parts) {
    $deduped = New-Object System.Collections.Generic.List[string]
    $seen = @{}
    foreach ($part in $Parts) {
        $trimmed = "$part".Trim()
        if ($trimmed -eq '') { continue }
        $key = Normalize-PathValue $trimmed
        if ($seen.ContainsKey($key)) { continue }
        $seen[$key] = $true
        $deduped.Add($trimmed)
    }
    [Environment]::SetEnvironmentVariable('Path', ($deduped -join ';'), $Scope)
}

function Add-PathEntries($Scope, $Entries) {
    if (-not $Entries) { return }
    $parts = Get-PathParts $Scope
    $newParts = @()
    foreach ($entry in @($Entries)) {
        $trimmed = "$entry".Trim()
        if ($trimmed -ne '') {
            $newParts += $trimmed
        }
    }
    $newParts += $parts
    Set-PathParts $Scope $newParts
}

function Remove-ManagedPathEntries($Scope, $Roots) {
    $parts = Get-PathParts $Scope
    if ($parts.Count -eq 0) { return }
    $kept = @()
    foreach ($part in $parts) {
        $managed = $false
        foreach ($root in @($Roots)) {
            if (Test-PathUnderRoot $part $root) {
                $managed = $true
                break
            }
        }
        if (-not $managed) {
            $kept += $part
        }
    }
    Set-PathParts $Scope $kept
}

function Remove-DirectoryIfExists($Path) {
    if ($Path -and (Test-Path -LiteralPath $Path)) {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Remove-FileIfExists($Path) {
    if ($Path -and (Test-Path -LiteralPath $Path)) {
        Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
    }
}

function Remove-DirectoryIfEmpty($Path) {
    if (-not $Path -or -not (Test-Path -LiteralPath $Path)) { return }
    try {
        $remaining = @(Get-ChildItem -LiteralPath $Path -Force -ErrorAction Stop)
        if ($remaining.Count -eq 0) {
            Remove-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
        }
    } catch {
        return
    }
}

function Add-UniquePathRoot([System.Collections.Generic.List[string]]$Roots, [string]$Path) {
    $trimmed = "$Path".Trim()
    if ($trimmed -eq '') { return }
    $normalized = Normalize-PathValue $trimmed
    foreach ($existing in $Roots) {
        if ((Normalize-PathValue $existing) -eq $normalized) {
            return
        }
    }
    $Roots.Add($trimmed)
}

function Add-VfoxHomeRoots([System.Collections.Generic.List[string]]$Roots, [string]$VfoxHomePath) {
    $home = "$VfoxHomePath".Trim()
    if ($home -eq '') { return }
    Add-UniquePathRoot $Roots $home
    Add-UniquePathRoot $Roots (Join-Path $home 'cache')
    Add-UniquePathRoot $Roots (Join-Path $home 'sdks')
    Add-UniquePathRoot $Roots (Join-Path $home 'path-shims')
}

function Read-ConfiguredVfoxHome([string]$ConfigFile) {
    if (-not (Test-Path -LiteralPath $ConfigFile)) { return '' }
    try {
        $config = Get-Content -LiteralPath $ConfigFile -Raw | ConvertFrom-Json
        if ($null -ne $config.PSObject.Properties['vfoxHome']) {
            return "$($config.vfoxHome)".Trim()
        }
    } catch {
        Write-Host "vfoxG cleanup: unable to read config: $($_.Exception.Message)"
    }
    return ''
}

function Read-HijackData([string]$HijackDataFile) {
    if (-not (Test-Path -LiteralPath $HijackDataFile)) { return $null }
    try {
        return Get-Content -LiteralPath $HijackDataFile -Raw | ConvertFrom-Json
    } catch {
        Write-Host "vfoxG cleanup: unable to read hijack data: $($_.Exception.Message)"
        return $null
    }
}

function Remove-VfoxHomeDataIfExists([string]$VfoxHomePath) {
    $home = "$VfoxHomePath".Trim()
    if ($home -eq '' -or -not (Test-Path -LiteralPath $home)) { return }

    foreach ($name in @(
        'cache',
        'plugin',
        'sdks',
        'path-shims',
        '.vfox.toml',
        'hijacked_paths.json',
        'gui-plugins-cache.json',
        'gui-system-sdks-cache.json',
        'gui-non-vfox-sdks.json'
    )) {
        $path = Join-Path $home $name
        if (Test-Path -LiteralPath $path -PathType Container) {
            Remove-DirectoryIfExists $path
        } else {
            Remove-FileIfExists $path
        }
    }

    Remove-DirectoryIfEmpty $home
}

$appData = [Environment]::GetFolderPath('ApplicationData')
$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
$userProfile = [Environment]::GetFolderPath('UserProfile')
$configRoot = Join-Path ([Environment]::GetFolderPath('ApplicationData')) $ProductName
$configFile = Join-Path $configRoot 'config.json'
$configuredVfoxHome = Read-ConfiguredVfoxHome $configFile
$vfoxRoot = Join-Path $appData $ProductName
$defaultVfoxHome = Join-Path $vfoxRoot 'vfox-home'
$vfoxHome = if ($configuredVfoxHome) { $configuredVfoxHome } else { $defaultVfoxHome }
$hijackFiles = New-Object System.Collections.Generic.List[string]
Add-UniquePathRoot $hijackFiles (Join-Path $vfoxHome 'hijacked_paths.json')
Add-UniquePathRoot $hijackFiles (Join-Path $defaultVfoxHome 'hijacked_paths.json')
$legacyVfoxHome = Join-Path $userProfile '.vfox'
Add-UniquePathRoot $hijackFiles (Join-Path $legacyVfoxHome 'hijacked_paths.json')
$managedPathRoots = New-Object System.Collections.Generic.List[string]

Add-UniquePathRoot $managedPathRoots $InstallDir
Add-UniquePathRoot $managedPathRoots (Join-Path $InstallDir 'core')
Add-VfoxHomeRoots $managedPathRoots $vfoxHome
Add-VfoxHomeRoots $managedPathRoots $defaultVfoxHome
Add-VfoxHomeRoots $managedPathRoots $legacyVfoxHome

foreach ($hijackFile in @($hijackFiles)) {
    $data = Read-HijackData $hijackFile
    if ($null -ne $data) {
        foreach ($property in @($data.PSObject.Properties)) {
            $entry = $property.Value
            if ($null -ne $entry.PSObject.Properties['UserPaths']) {
                Add-PathEntries 'User' @($entry.UserPaths)
            }
            if ($null -ne $entry.PSObject.Properties['MachinePaths']) {
                Add-PathEntries 'Machine' @($entry.MachinePaths)
            }
            if ($null -ne $entry.PSObject.Properties['ShimDir']) {
                Add-UniquePathRoot $managedPathRoots "$($entry.ShimDir)"
            }
        }
    }
}

Remove-ManagedPathEntries 'User' $managedPathRoots
Remove-ManagedPathEntries 'Machine' $managedPathRoots
[Environment]::SetEnvironmentVariable('VFOX_HOME', $null, 'User')
[Environment]::SetEnvironmentVariable('VFOX_HOME', $null, 'Machine')
[Environment]::SetEnvironmentVariable('__VFOX_SHELL', $null, 'User')
[Environment]::SetEnvironmentVariable('__VFOX_SHELL', $null, 'Machine')

foreach ($hijackFile in @($hijackFiles)) {
    if (Test-Path -LiteralPath $hijackFile) {
        Remove-Item -LiteralPath $hijackFile -Force -ErrorAction SilentlyContinue
    }
}

foreach ($home in @($vfoxHome, $defaultVfoxHome, $legacyVfoxHome)) {
    $shimPath = Join-Path $home 'path-shims'
    Remove-DirectoryIfExists $shimPath
}

Remove-DirectoryIfExists (Join-Path $appData $ProductExecutable)
Remove-DirectoryIfExists (Join-Path $localAppData $ProductExecutable)
Remove-DirectoryIfExists (Join-Path $localAppData $ProductName)

if ($RemoveSdkData) {
    Remove-DirectoryIfExists $vfoxRoot
    if ($configuredVfoxHome -and -not (Test-PathUnderRoot $configuredVfoxHome $vfoxRoot)) {
        Remove-VfoxHomeDataIfExists $configuredVfoxHome
    }
    Remove-VfoxHomeDataIfExists $legacyVfoxHome
}

try {
    Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
    $result = [UIntPtr]::Zero
    [Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result) | Out-Null
} catch {
    Write-Host "vfoxG cleanup: unable to broadcast environment change: $($_.Exception.Message)"
}
