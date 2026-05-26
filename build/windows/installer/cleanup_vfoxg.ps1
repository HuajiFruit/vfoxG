param(
    [string]$InstallDir,
    [string]$ProductName,
    [string]$ProductExecutable
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

$appData = [Environment]::GetFolderPath('ApplicationData')
$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
$userProfile = [Environment]::GetFolderPath('UserProfile')
$vfoxRoot = Join-Path $appData $ProductName
$vfoxHome = Join-Path $vfoxRoot 'vfox-home'
$hijackFile = Join-Path $vfoxHome 'hijacked_paths.json'
$shimDir = Join-Path $vfoxHome 'path-shims'
$sdksDir = Join-Path $vfoxHome 'sdks'
$legacyVfoxHome = Join-Path $userProfile '.vfox'
$legacyShimDir = Join-Path $legacyVfoxHome 'path-shims'
$legacySdksDir = Join-Path $legacyVfoxHome 'sdks'

$managedPathRoots = @(
    $InstallDir,
    (Join-Path $InstallDir 'core'),
    $shimDir,
    $sdksDir,
    $vfoxHome,
    $legacyShimDir,
    $legacySdksDir,
    $legacyVfoxHome
)

if (Test-Path -LiteralPath $hijackFile) {
    try {
        $data = Get-Content -LiteralPath $hijackFile -Raw | ConvertFrom-Json
        foreach ($property in @($data.PSObject.Properties)) {
            $entry = $property.Value
            if ($null -ne $entry.PSObject.Properties['UserPaths']) {
                Add-PathEntries 'User' @($entry.UserPaths)
            }
            if ($null -ne $entry.PSObject.Properties['MachinePaths']) {
                Add-PathEntries 'Machine' @($entry.MachinePaths)
            }
            if ($null -ne $entry.PSObject.Properties['ShimDir']) {
                $managedPathRoots += "$($entry.ShimDir)"
            }
        }
    } catch {
        Write-Host "vfoxG cleanup: unable to read hijack data: $($_.Exception.Message)"
    }
}

Remove-ManagedPathEntries 'User' $managedPathRoots
Remove-ManagedPathEntries 'Machine' $managedPathRoots

Remove-DirectoryIfExists (Join-Path $appData $ProductExecutable)
Remove-DirectoryIfExists (Join-Path $localAppData $ProductExecutable)
Remove-DirectoryIfExists $vfoxRoot
Remove-DirectoryIfExists (Join-Path $localAppData $ProductName)
Remove-DirectoryIfExists $legacyVfoxHome

try {
    Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition '[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)] public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);'
    $result = [UIntPtr]::Zero
    [Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result) | Out-Null
} catch {
    Write-Host "vfoxG cleanup: unable to broadcast environment change: $($_.Exception.Message)"
}
