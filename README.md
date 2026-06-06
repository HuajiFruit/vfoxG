# vfoxG

[中文文档](READMEcn.md)

<p align="center">
  <img src="build/appicon.png" alt="vfoxG" width="128" />
</p>

<p align="center">
  <strong>vfox GUI Manager · Wails + Vue 3</strong>
</p>

<p align="center">
  Manage vfox SDKs, plugins, system SDKs, PATH integration, and environment export/import from a desktop GUI.
</p>

---

## Screenshot

![vfoxG screenshot](image.png)

## Downloads

Download published builds from [GitHub Releases](https://github.com/HuajiFruit/vfoxG/releases).

vfoxG is built with Wails. The verified release workflow currently builds Windows installers:

| Platform | Artifact |
| --- | --- |
| Windows amd64 | `vfoxG-windows-amd64-installer.exe` |
| Windows 386 | `vfoxG-windows-386-installer.exe` |

Other desktop platforms can be built after the matching vfox core binaries and platform-specific behavior are verified.

## Features

- SDK management: view, install, uninstall, switch, and unuse vfox-managed SDK versions.
- Custom SDKs: add system-installed SDKs, detect versions from executables, and switch them through the same managed entrypoints.
- Plugin marketplace: browse available vfox plugins, add plugins, remove plugins, and keep custom SDK paths when needed.
- Release-aware version search: versions with no published releases are shown as available to retry instead of locking the SDK.
- PATH integration: add or remove vfox from user PATH, and manage SDK command overrides where supported.
- Windows compatibility: handle App Execution Alias conflicts, junctions, hidden PowerShell helper windows, and installer cleanup.
- Download directory migration: move the vfox home/download location with progress feedback and a modal confirmation flow.
- Environment sync: export the current SDK environment and import it on another machine.
- Bilingual UI and docs: Chinese and English resources are kept in parallel.

## Documentation

| Topic | English | Chinese |
| --- | --- | --- |
| Documentation index | [docs/README.md](docs/README.md) | [docs/README.zh-CN.md](docs/README.zh-CN.md) |
| Architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | [docs/ARCHITECTURE.zh-CN.md](docs/ARCHITECTURE.zh-CN.md) |
| Development | [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | [docs/DEVELOPMENT.zh-CN.md](docs/DEVELOPMENT.zh-CN.md) |
| Release | [docs/RELEASE.md](docs/RELEASE.md) | [docs/RELEASE.zh-CN.md](docs/RELEASE.zh-CN.md) |
| Code style | [docs/CODE_STYLE.en.md](docs/CODE_STYLE.en.md) | [docs/CODE_STYLE.md](docs/CODE_STYLE.md) |
| Frontend | [frontend/README.md](frontend/README.md) | [frontend/README.zh-CN.md](frontend/README.zh-CN.md) |
| Build assets | [build/README.md](build/README.md) | [build/README.zh-CN.md](build/README.zh-CN.md) |

## Architecture Overview

```text
+------------------------------+
| Frontend: Vue 3 + TypeScript |
| components / composables     |
| services / i18n / styles     |
+--------------+---------------+
               |
               | Wails generated bindings
               v
+--------------+---------------+
| Backend: Go package main     |
| app_facade_* thin API layer  |
| sdk / plugin / sync / config |
| parser / path / platform     |
+--------------+---------------+
               |
               | command execution
               v
+--------------+---------------+
| vfox CLI core                |
| bundled in release builds    |
+------------------------------+
```

The backend is intentionally split into small atomic files while staying in `package main` to keep Wails bindings stable. Public Wails methods live in `app_facade_*.go`; domain logic lives in focused files such as `sdk_use.go`, `plugin_remove.go`, `migration_run.go`, and `sync_import_apply.go`.

The frontend follows a component/composable/service direction:

```text
components -> composables -> services -> frontend/wailsjs
```

Components handle rendering and user events, composables hold UI state and workflows, and services are the only layer that imports generated Wails bindings.

## Runtime Core

Release builds download and bundle the Windows vfox core binaries declared in `.github/workflows/release.yml`.

For local development, put the matching vfox executable under `core/`:

```text
core/
  windows/
    x86_64/vfox.exe
    x86/vfox.exe
```

`core/` is ignored by Git so local binaries are not committed.

## Development

Requirements:

| Tool | Version |
| --- | --- |
| Go | 1.23+ |
| Node.js | 22+ |
| Wails CLI | v2 |
| NSIS | 3.x, Windows installer only |

Install frontend dependencies:

```bash
npm --prefix frontend install
```

Run the desktop app:

```bash
wails dev
```

Verify the project:

```bash
go test ./...
npm --prefix frontend run build
```

Build locally:

```bash
wails build -clean
```

Build a Windows amd64 installer:

```bash
wails build -platform windows/amd64 -nsis -clean
```

## Project Structure

```text
vfoxG/
  app.go, app_lifecycle.go       App state and lifecycle
  app_facade_*.go                Wails API facade methods
  model_*.go                     DTOs shared with frontend bindings
  config_*.go                    app config, VFOX_HOME, download path
  vfox_*.go                      vfox executable, command, env, progress, task lock
  parse_*.go                     pure parsers for vfox output
  sdk_*.go                       SDK list/detail/install/use/custom SDK logic
  plugin_*.go                    plugin marketplace, state, add/remove, cache
  system_*.go                    system SDK definitions, scanning, cache
  path_*.go                      shared PATH comparison and managed-root helpers
  migration_*.go                 download directory migration and repair
  sync_*.go                      SDK environment export/import
  windows_*.go                   Windows-specific PATH, shim, junction, elevation logic
  unix_*.go                      Unix-specific PATH and symlink logic
  frontend/                      Vue 3 frontend
  build/                         icons, manifests, installer scripts
  docs/                          bilingual project documentation
  .github/workflows/             release pipeline
```

## Release Process

Releases are created by pushing a `v*` tag:

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

The GitHub Actions workflow downloads vfox core binaries, builds Windows installers, and uploads release assets.

## Acknowledgments

Thanks to the [vfox](https://github.com/version-fox/vfox) project for the cross-platform version management engine that powers vfoxG.

vfoxG is an independent third-party GUI and is not affiliated with or endorsed by the vfox project.
