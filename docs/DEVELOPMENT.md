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

- Keep the public method in the matching `internal/app/app_facade_*.go` file.
- Keep exported DTOs in `internal/model/`.
- Rebuild or run the app so Wails regenerates bindings if needed.
- Commit generated binding updates only when the API surface actually changed.

## Backend Workflow

The backend is split under `internal/`:

1. Keep `main.go` as the Wails startup and binding entrypoint only.
2. Add or change public Wails methods in `internal/app/app_facade_*.go`.
3. Put stateful workflows in focused `internal/app/*` files, such as `sdk_use.go`, `plugin_remove.go`, or `sync_import_apply.go`.
4. Keep exported DTOs in `internal/model/`.
5. Keep pure vfox output parsing in `internal/parser/` with table-driven tests.
6. Keep Windows behavior in `internal/app/windows_*.go` and Unix behavior in `internal/app/unix_*.go`.

Do not put backend business logic in the repository root. Do not expand `internal/app/app.go` with new workflows; it should remain app state oriented.

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
