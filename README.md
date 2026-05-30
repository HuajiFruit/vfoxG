# vfoxG

[中文文档](READMEcn.md)

<p align="center">
  <img src="build/appicon.png" alt="vfoxG" width="128" />
</p>

<p align="center">
  <strong>vfox GUI Manager · Wails + Vue 3</strong>
</p>

<p align="center">
  Manage vfox SDKs and custom system SDKs in one place. Install, remove, and switch versions without leaving the GUI.
</p>

---

## Screenshots

![vfoxG Screenshot](build/screenshot.png)

## Downloads

Download the latest release from [GitHub Releases](https://github.com/HuajiFruit/vfoxG/releases).

vfoxG is built with Wails and can be packaged for multiple desktop platforms. At the moment, only the Windows release builds have been verified and published:

| Platform | Artifact |
| --- | --- |
| Windows amd64 | `vfoxG-windows-amd64-installer.exe` |
| Windows 386 | `vfoxG-windows-386-installer.exe` |

## Features

### SDK Management

- View, install, uninstall, and switch vfox-managed SDK versions.
- Add system-installed SDKs as custom SDK entries.
- Detect common system SDKs automatically.
- Keep SDK exposure stable through `~/.vfox/sdks/{name}` links.

### Plugin Marketplace

- Browse available vfox plugins.
- Add plugins from the GUI.
- Search installable SDK versions with progress feedback.

### System Integration

- Manage PATH integration from the app.
- Resolve Windows App Execution Alias conflicts.
- Request elevation only when system-level changes are needed.

## Architecture

```text
+------------------------------------------+
|           Frontend (Vue 3)               |
|  SdkManager.vue | PluginMarket | Settings |
+------------------------------------------+
|        Wails v2 Bridge (RPC)             |
+------------------------------------------+
|           Backend (Go)                   |
|  app.go - SDK mgmt, version switch, PATH |
+------------------------------------------+
|         vfox CLI (core/)                 |
|  bundled or locally provided runtime      |
+------------------------------------------+
```

## Runtime Core

Release builds bundle the required Windows vfox core binaries. Packaging other platforms is possible, but those packages need matching vfox core binaries and platform-specific verification before they are published.

For local development, place vfox in the matching `core/` directory:

```text
core/
  windows/
    x86_64/vfox.exe
    x86/vfox.exe
```

The `core/` directory is ignored by Git.

## Development

### Requirements

| Tool | Version |
| --- | --- |
| Go | 1.23+ |
| Node.js | 22+ |
| Wails CLI | v2 |
| NSIS | 3.x, Windows installer only |

### Run

```bash
cd frontend
npm install
cd ..
wails dev
```

### Test

```bash
go test ./...
npm --prefix frontend run build
```

### Build Locally

```bash
wails build -clean
```

Windows installer:

```bash
wails build -platform windows/amd64 -nsis -clean
```

Other Wails targets can be built with the appropriate platform toolchain and bundled vfox core, but they are not currently covered by the verified release workflow.

## Release Process

Releases are built by GitHub Actions when a `v*` tag is pushed.

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

The release workflow currently downloads Windows vfox core binaries, builds Windows installers, and uploads release assets automatically. Other platform packages may be added later after they are tested.

## Project Structure

```text
vfoxG/
  app.go                  Go backend core
  app_windows.go          Windows platform integration
  frontend/               Vue 3 frontend
  build/windows/          Windows installer assets
  .github/workflows/      GitHub Actions release pipeline
  wails.json              Wails project config
```

## Acknowledgments

Special thanks to the [vfox](https://github.com/version-fox/vfox) project for providing the cross-platform version management engine used by vfoxG.

vfoxG is an independent third-party GUI. It is not affiliated with or endorsed by the vfox project.
