# Development Guide

[中文](DEVELOPMENT.zh-CN.md)

This guide describes how to set up, run, verify, and change vfoxG locally.

## Requirements

| Tool | Version | Notes |
| --- | --- | --- |
| Go | 1.23+ | `go.mod` declares Go 1.23.0. |
| Node.js | 22+ | Matches the release workflow. |
| npm | Bundled with Node | Used for the Vite frontend. |
| Wails CLI | v2 | Used for `wails dev` and `wails build`. |
| NSIS | 3.x | Required only when building Windows installers. |

Install Wails if it is not available:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## First Setup

Install frontend dependencies from the repository root:

```bash
npm --prefix frontend install
```

For local runtime testing, place the vfox core binary under `core/`:

```text
core/
  windows/
    x86_64/vfox.exe
    x86/vfox.exe
```

Release builds download this directory automatically, but local development expects you to provide it. The `core/` directory is ignored by Git.

## Run Locally

Start Wails development mode:

```bash
wails dev
```

Wails starts the Go backend and the Vite frontend together. The frontend dev server command is configured in `wails.json`.

## Verification Commands

Run backend tests:

```bash
go test ./...
```

Build and type-check the frontend:

```bash
npm --prefix frontend run build
```

Check whitespace in Git-tracked changes:

```bash
git diff --check
```

For Go-only changes, run `gofmt` on changed Go files before testing. For frontend changes, keep TypeScript and Vue compilation green through the frontend build command.

## Wails Bindings

Generated frontend bindings live under `frontend/wailsjs/`. They are generated from exported Go methods and model structs.

When changing exported `App` methods or DTOs:

- Keep the public method in the matching `app_facade_*.go` file.
- Keep model structs in `model_*.go`.
- Rebuild or run the app so Wails regenerates bindings if needed.
- Commit generated binding updates only when the API surface actually changed.

## Backend Workflow

The backend is split by responsibility while remaining in `package main`:

1. Add or change the public method in an `app_facade_*.go` file.
2. Put behavior in the focused domain file, such as `sdk_use.go`, `plugin_remove.go`, or `sync_import_apply.go`.
3. Keep parsing in `parse_*.go` pure and covered by table-driven tests.
4. Keep file I/O in config, storage, cache, or platform files.
5. Keep Windows behavior in `windows_*.go` and Unix behavior in `unix_*.go`.

Do not expand `app.go` with new business logic. It should remain app state and lifecycle oriented.

## Frontend Workflow

Frontend source lives in `frontend/src/`:

1. Components render state and emit user events.
2. Composables own UI state and async workflows.
3. Services call generated Wails bindings.
4. Text goes through `frontend/src/i18n/`.
5. Shared styling belongs in `frontend/src/styles/`.

When adding a user-facing string, update both `frontend/src/i18n/en.ts` and `frontend/src/i18n/zh.ts`, and keep key parity through `frontend/src/i18n/keys.ts`.

## Debugging Tips

- Use the terminal dock and task toast output for long vfox operations.
- If SDK switching works in vfox CLI but not in the app, inspect the managed entrypoint path and PATH override state.
- On Windows, check App Execution Alias conflicts when a command opens the wrong executable.
- If migration behaves unexpectedly, inspect the migration progress event and the old/new VFOX_HOME values before changing repair logic.
- If uninstall leaves command residue, inspect Windows shim files and override metadata before changing installer cleanup.

## Documentation Updates

Update docs in the same change when behavior changes:

| Change | Docs to check |
| --- | --- |
| New backend module or file pattern | `docs/ARCHITECTURE.md`, `docs/ARCHITECTURE.zh-CN.md` |
| New developer command | `docs/DEVELOPMENT.md`, `docs/DEVELOPMENT.zh-CN.md` |
| Release workflow change | `docs/RELEASE.md`, `docs/RELEASE.zh-CN.md` |
| Naming, comments, file split rules | `docs/CODE_STYLE.en.md`, `docs/CODE_STYLE.md` |
| User-visible feature | root `README.md` and `READMEcn.md` |
