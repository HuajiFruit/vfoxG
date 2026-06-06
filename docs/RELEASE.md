# Release Guide

[中文](RELEASE.zh-CN.md)

vfoxG releases are created by GitHub Actions when a tag matching `v*` is pushed.

## Current Release Scope

The verified workflow builds Windows installers:

| Platform | Artifact |
| --- | --- |
| Windows amd64 | `vfoxG-windows-amd64-installer.exe` |
| Windows 386 | `vfoxG-windows-386-installer.exe` |

The workflow runs on `windows-2025`, installs Go, Node.js, NSIS, and Wails, downloads vfox core binaries, then builds and uploads installer assets.

## vfox Core Version

The bundled vfox version is controlled by the `VFOX_VERSION` environment variable in `.github/workflows/release.yml`.

Before updating it:

- Confirm the upstream vfox release exists.
- Confirm both Windows archives exist:
  - `vfox_<version>_windows_x86_64.zip`
  - `vfox_<version>_windows_i386.zip`
- Run local smoke tests when possible.
- Mention the core version change in the release notes.

## Pre-Release Checklist

Run these checks before tagging:

```bash
go test ./...
npm --prefix frontend run build
git diff --check
```

Manual checks:

- App launches without a console window for installer/uninstaller helper operations.
- SDK list, detail, version search, install, uninstall, use, and unuse work.
- SDKs with no published releases show a retryable "no release version" state instead of locking the action.
- Custom SDK add, detect, use, and remove work.
- Plugin add and remove keep or delete custom SDK paths according to user choice.
- Download path migration confirmation is centered like other floating windows.
- Uninstall removes app data, shims, and PATH/override residue expected by the installer cleanup design.
- English and Chinese UI text both render correctly.

## Tagging

Create and push an annotated tag:

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

GitHub Actions creates or updates the matching GitHub Release and uploads files from `release_staging/`.

## Installer Assets

Windows installer files live under `build/windows/installer/`:

| File | Purpose |
| --- | --- |
| `project.nsi` | Main Windows installer script. |
| `project_386.nsi` | 32-bit installer script. |
| `cleanup_vfoxg.ps1` | Cleanup helper used by uninstall logic. |
| `wails_tools.nsh` | Installer helper macros and Wails integration. |

The installer should not leave visible PowerShell windows during uninstall. If cleanup behavior changes, verify it with a real installed build instead of only running from source.

## Local Build Commands

Generic local build:

```bash
wails build -clean
```

Windows amd64 installer:

```bash
wails build -platform windows/amd64 -nsis -clean
```

Windows 386 installer follows the release workflow because it calls NSIS directly with `project_386.nsi`.

## Release Notes

Release notes should include:

- User-visible fixes and features.
- SDK/plugin/PATH/migration behavior changes.
- vfox core version changes.
- Known platform limitations.
- Upgrade or uninstall notes when cleanup behavior changes.

Avoid listing internal refactors unless they affect maintainability, testing, or future contributor workflow.
