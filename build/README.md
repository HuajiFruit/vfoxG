# Build Assets

[中文](README.zh-CN.md)

This directory contains Wails build assets, platform manifests, and Windows installer scripts for vfoxG.

## Directory Layout

```text
build/
  appicon.png                   Source icon used by Wails
  screenshot.png                Optional screenshot asset
  darwin/
    Info.plist                  macOS production plist
    Info.dev.plist              macOS development plist
  windows/
    icon.ico                    Windows application icon
    info.json                   Windows version/resource metadata
    wails.exe.manifest          Windows app manifest
    installer/
      project.nsi               Main Windows installer script
      project_386.nsi           32-bit Windows installer script
      cleanup_vfoxg.ps1         Uninstall cleanup helper
      wails_tools.nsh           Wails/NSIS helper macros
```

## Windows Installer

The verified release workflow builds Windows installers. The amd64 installer is built by Wails with NSIS enabled:

```bash
wails build -platform windows/amd64 -nsis -clean
```

The 386 installer is built by the release workflow with `project_386.nsi` and NSIS directly.

Installer cleanup behavior matters because vfoxG creates managed SDK entrypoints, Windows shims, and PATH override metadata. When changing installer files, verify a real install and uninstall flow.

## Hidden Helper Windows

Uninstall and cleanup helpers should not show a visible PowerShell window. If installer scripts launch PowerShell, use the existing hidden-window approach and confirm it with an installed build.

## vfox Core

Release builds download vfox core binaries into `core/` before packaging. The `core/` directory is intentionally outside this `build/` directory and is ignored by Git.

Do not commit downloaded vfox binaries.

## Safe Editing Rules

- Keep Wails-generated default files unless a product behavior requires changes.
- Document installer behavior changes in `docs/RELEASE.md` and `docs/RELEASE.zh-CN.md`.
- Test both install and uninstall when touching `build/windows/installer/`.
- Avoid changing app icons or manifests together with unrelated code behavior.
