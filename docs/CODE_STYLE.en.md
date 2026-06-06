# vfoxG Code Style and Atomic Split Plan

[中文](CODE_STYLE.md)

This document defines maintainability rules for vfoxG. The goal is not to move files for cosmetic reasons; the goal is to make each file understandable, testable, and changeable on its own.

## Maintainability Goals

- One file owns one atomic responsibility.
- One function performs one behavioral step.
- One module exposes a small stable entry surface.
- State changes must be testable.
- Platform differences must be isolated.
- The frontend must not mutate system state directly.
- The backend must not own UI presentation details.
- New behavior must not be added to already oversized files.

Recommended thresholds:

| Type | Recommended limit | Action after exceeding it |
| --- | ---: | --- |
| Go normal file | 250 lines | Split into a smaller atomic file. |
| Go platform file | 300 lines | Split by PATH, process, file-system, or elevation behavior. |
| Go function | 60 lines | Split into validate, plan, apply, parse, or helper functions. |
| Vue page component | 300 lines | Extract child components or composables. |
| Vue child component | 220 lines | Split presentation and behavior. |
| Composable/service | 180 lines | Split by state domain. |
| CSS file | 400 lines | Split tokens, layout, components, and views. |

These are review thresholds, not compiler rules.

## Dependency Direction

Backend direction:

```text
app facade
  -> usecase/service
    -> domain model/parser
    -> platform adapter
    -> storage
```

Forbidden reverse dependencies:

- Parsers must not depend on `App`.
- Models must not depend on Wails runtime code.
- Platform adapters must not depend on UI events.
- Storage must not run vfox commands.
- Command executors must not parse domain-specific results.

Frontend direction:

```text
views/components
  -> composables
    -> services
      -> wailsjs generated bindings
```

Forbidden:

- Importing many Wails APIs directly inside components.
- Services depending on component state.
- Composables mutating DOM or writing styles.
- i18n text scattered through components.

## Current Structure

The current backend split keeps all Go files in `package main`:

```text
app_lifecycle.go               App struct, startup, event helpers
app_facade_*.go                Public Wails methods
model_*.go                     Shared DTOs
config_*.go                    App config, VFOX_HOME, download path
vfox_*.go                      vfox executable, command, env, progress, task lock
parse_*.go                     Pure vfox output parsers
sdk_*.go                       SDK inventory, detail, install, use, custom SDK behavior
plugin_*.go                    Plugin market, state, add/remove, cache
system_*.go                    System SDK scan, version probing, cache
path_*.go                      Shared path comparison and managed roots
migration_*.go                 Download path migration, copy, progress, repair
sync_*.go                      SDK environment export/import
windows_*.go                   Windows PATH, shim, junction, elevation, override metadata
unix_*.go                      Unix PATH, symlink, executable, override behavior
```

If a future phase splits packages, use this direction:

```text
internal/
  appfacade/
  config/
  vfoxcli/
  sdk/
  plugin/
  systemsdk/
  pathenv/
  migration/
  sync/
  platform/
```

Do not split packages until tests and Wails bindings can absorb the change safely.

## Backend Rules

### App Facade

`app_facade_*.go` files may only:

- Trim and validate input.
- Acquire task locks.
- Call focused use-case functions.
- Emit required events.

They must not:

- Write files.
- Run vfox commands directly when a domain helper exists.
- Parse vfox output.
- Mutate PATH directly.
- Build complex paths.

### vfox Command

`vfox_command.go` owns short command execution:

- Context timeout.
- Hidden window settings.
- Cleaned environment.
- Combined output.
- ANSI cleanup.

`vfox_progress.go` owns long-running command output:

- stdout/stderr scanners.
- Progress line events.
- No-release output normalization.
- Done and exit-error events.

Neither file should refresh SDK lists, write caches, or mutate PATH.

### Parsers

Parser files must be pure:

- Input: string.
- Output: struct, slice, or string.
- No file reads.
- No vfox calls.
- No events.

Every parser should have table-driven tests.

### SDK Workflows

Use separate files for separate state flows:

| File | Responsibility |
| --- | --- |
| `sdk_install.go` | `vfox install` and post-install refresh events. |
| `sdk_uninstall.go` | Version uninstall and active-version cleanup. |
| `sdk_use.go` | Use/unuse version flow and managed entrypoint refresh. |
| `sdk_runtime_root.go` | Resolve actual runtime root from vfox info/path data. |
| `sdk_detail.go` | Build SDK detail data before UI-level presentation. |
| `sdk_custom_registry.go` | Custom SDK JSON registry read/write. |
| `sdk_custom_detect.go` | Detect versions from executable commands. |
| `sdk_custom_use.go` | Activate custom SDKs through junctions or symlinks. |

Registry code must not create junctions. Detect code must not write JSON. Use code must name each step explicitly: validate, lock, clear conflicting state, run command, resolve root, refresh entrypoint, refresh override, emit events.

### Plugin Workflows

| File | Responsibility |
| --- | --- |
| `plugin_market.go` | Available plugin list and refresh. |
| `plugin_state.go` | Added plugin state and plugin directory scan. |
| `plugin_add.go` | Add plugin flow. |
| `plugin_remove.go` | Remove plugin flow. |
| `plugin_description_cache.go` | GUI plugin cache read/write. |

Plugin removal should keep "custom SDK preservation", "PATH override restore", and "plugin data deletion" in separate helpers.

### System SDK Scan

| File | Responsibility |
| --- | --- |
| `system_defs.go` | SDK definition table. |
| `system_scan.go` | Scan orchestration and filtering. |
| `system_version.go` | Run version commands and reject invalid output. |
| `system_cache.go` | GUI cache read/write. |

Scanning must not mutate global process environment.

### PATH and Override

Shared helpers belong in `path_*.go`. Windows-specific behavior belongs in `windows_*.go`. Unix-specific behavior belongs in `unix_*.go`.

Windows override metadata code only reads and writes JSON. It must not elevate, mutate PATH, or remove files.

### Migration

Migration must be separated into detection, copy, progress, and repair:

| File | Responsibility |
| --- | --- |
| `migration_detect.go` | Detect whether migration is needed. |
| `migration_run.go` | Orchestrate migration. |
| `migration_copy*.go` | Copy files, directories, and links without overwriting. |
| `migration_progress.go` | Track and emit progress. |
| `migration_repair.go` | Repair SDK entrypoints and path overrides after migration. |

Migration should plan before applying. Failed apply steps must not silently corrupt the new config.

### Sync Import/Export

Parse and format functions must be pure. Apply functions may call SDK/plugin operations.

| File | Responsibility |
| --- | --- |
| `sync_export_collect.go` | Collect current SDK environment. |
| `sync_export_format.go` | Format export text. |
| `sync_import_parse.go` | Parse import text. |
| `sync_import_apply.go` | Apply import targets. |

## Frontend Rules

Target direction:

```text
components -> composables -> services -> wailsjs
```

### Components

Components should:

- Render props and composable state.
- Emit user actions.
- Keep template logic readable.
- Use i18n keys for user-facing text.

Components should not:

- Import Wails bindings directly.
- Own multi-step business workflows.
- Mutate global state outside composables.
- Contain large API orchestration blocks.

### Composables

Composables should:

- Own `ref`, `computed`, loading, and error state.
- Call services.
- Expose clear actions to components.
- Handle async workflow cleanup in `finally`.

Composables should not:

- Write CSS.
- Depend on DOM details.
- Duplicate the same state under two names.

### Services

Services should:

- Import generated Wails bindings.
- Provide typed API wrappers.
- Normalize error handling where useful.

Services should not:

- Hold Vue refs.
- Import components.
- Translate UI text.

## Naming Rules

Names must describe business meaning before type or state. Avoid vague names such as `data`, `info`, `list`, `item`, `tmp`, `obj`, and `res` outside tiny local scopes.

| Situation | Rule | Example |
| --- | --- | --- |
| Boolean | Start with `is`, `has`, `can`, `should`, `allow`, or `enable`. | `isInstalled`, `hasReleaseVersion`, `canUseSdk` |
| Loading state | Include the action. | `loadingSdkList`, `savingSettings` |
| Count | Use `Count`. | `versionCount` |
| Index | Use `Index`; reserve `i/j` for tiny loops. | `selectedVersionIndex` |
| File path | Use `Path`; directories use `Dir`; roots use `Root`. | `sdkPath`, `downloadDir`, `vfoxRoot` |
| Version | Use full `Version`, not `ver` in business code. | `targetVersion` |
| SDK | Include SDK meaning. | `sdkName`, `selectedSdkName` |
| Plugin | Include plugin meaning. | `pluginName`, `pluginDir` |
| Slice/array | Use plural nouns. | `sdks`, `versions`, `plugins` |
| Map | Use `By` or `Map` to describe indexing. | `sdkByName`, `versionStatusMap` |
| Set | Use `Set` suffix. | `installedVersionSet` |
| Error | Use `err` locally; add business prefix when preserving multiple errors. | `installErr`, `migrationErr` |

If two variables can only be distinguished by `1`, `2`, `New`, or `Old`, first check whether they are the same state. Identical values such as `image` and `image1` must be merged into one business-named variable such as `sdkIconImage`.

## Comment Rules

Comments explain why, not what.

Avoid:

```go
// Remove file.
os.Remove(path)
```

Prefer:

```go
// vfox can leave a stale junction after switching versions.
// Refresh it so terminal shims resolve the newly selected SDK.
a.removeJunctionIfExists(sdkLinkPath)
```

Required comment areas:

- Windows elevation and hidden-window behavior.
- User PATH vs machine PATH differences.
- Junction/symlink relationship with vfox runtime roots.
- Compatibility with old JSON/TOML/CLI output.
- Destructive data removal and uninstall flows.
- Migration failure guarantees.

Public Go methods use English comments and start with the method name.

## Test Rules

- Test files follow the atomic source file name.
- Parsers use table-driven tests.
- File-system tests use `t.TempDir()`.
- Environment tests use `t.Setenv()`.
- Windows-only tests use `*_windows_test.go`.
- Tests must not depend on the user's real SDK state.
- Tests must not mutate the real PATH.

## Review Gates

Every code split or feature change should satisfy:

- No unintended behavior change.
- `go test ./...` passes for backend changes.
- `npm --prefix frontend run build` passes for frontend changes.
- New files have one clear responsibility.
- Large files do not grow further.
- New or existing tests cover the changed behavior.
- Wails bindings have no unrelated churn.

## Forbidden Patterns

- Growing `app.go`, platform monoliths, `SdkManager.vue`, or `PluginMarket.vue` with new unrelated logic.
- Creating a file that mixes parser, I/O, and state mutation.
- Letting components call multiple Wails APIs and assemble a backend workflow.
- Reading files or emitting events from parser code.
- Writing styles from services or composables.
- Moving many files at once without reducing responsibility.
- Splitting SDK, PATH, migration, or uninstall logic without tests.
