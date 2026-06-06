# Architecture

[中文](ARCHITECTURE.zh-CN.md)

vfoxG is a Wails desktop application. The frontend is Vue 3 and TypeScript. The backend is Go. The backend delegates version management to the vfox CLI core and wraps it with GUI-oriented state, validation, migration, and platform integration.

## Runtime Layers

```text
Vue components
  -> composables
    -> frontend services
      -> Wails generated bindings
        -> Go app facade
          -> SDK/plugin/sync/config use cases
            -> parser, storage, path, platform, vfox command helpers
              -> vfox CLI core and operating system
```

Each layer has a narrow responsibility:

| Layer | Responsibility |
| --- | --- |
| Components | Render UI, receive props, emit user actions. |
| Composables | Own Vue state, loading flags, task orchestration, and user workflow state. |
| Services | Wrap generated Wails bindings and normalize frontend API calls. |
| App facade | Expose stable Wails methods and delegate to focused backend functions. |
| Use cases | Implement SDK, plugin, sync, migration, config, and PATH workflows. |
| Parsers | Convert vfox command output into typed data without side effects. |
| Platform adapters | Isolate Windows and Unix behavior. |

## Backend Shape

The backend currently stays in `package main`. This keeps Wails binding generation simple while still allowing atomic files.

### Facade Files

`app_facade_*.go` files are the public API surface used by the frontend:

| File | Public area |
| --- | --- |
| `app_facade_sdk.go` | SDK list, detail, version install/use/unuse, custom SDK operations. |
| `app_facade_plugin.go` | Plugin marketplace, added plugins, plugin removal. |
| `app_facade_path.go` | PATH integration, override checks, hijack/restore operations. |
| `app_facade_settings.go` | Download path, migration choices, platform information. |
| `app_facade_sync.go` | SDK environment export, preview, and import. |
| `app_facade_system.go` | Cached and active system SDK scanning. |
| `app_facade_vfox.go` | Raw vfox command and progress command bridge. |

Facade functions should stay thin: validate input, acquire task locks when needed, call the focused implementation, and emit required events.

### Domain and Workflow Files

| Pattern | Responsibility |
| --- | --- |
| `model_*.go` | DTOs and model structs shared with Wails bindings. |
| `config_*.go` | App config, download path, VFOX_HOME, JSON persistence. |
| `vfox_*.go` | vfox executable lookup, command execution, clean environment, progress output, task lock. |
| `parse_*.go` | Pure parsers for installed SDKs, details, current version, search output, and version normalization. |
| `sdk_*.go` | SDK inventory, detail, install, uninstall, use, runtime root, and custom SDK registry/use/detect. |
| `plugin_*.go` | Marketplace loading, added plugin state, plugin add/remove, description cache. |
| `system_*.go` | System SDK definitions, scan orchestration, version probing, cache. |
| `path_*.go` | Cross-platform path comparison and managed root helpers. |
| `migration_*.go` | Download directory migration, no-overwrite copy, progress, repair. |
| `sync_*.go` | SDK environment export/import collection, parsing, formatting, and apply logic. |
| `windows_*.go` | Windows PATH, shims, junctions, elevation helpers, override metadata. |
| `unix_*.go` | Unix PATH profile blocks, symlinks, executable checks, override behavior. |

## Frontend Shape

```text
frontend/src/
  App.vue
  app/
    navigation.ts
  components/
    app/
    common/
    plugin/
    sdk/
    settings/
    sync/
  composables/
  services/
  i18n/
  styles/
```

### Component Groups

| Directory | Responsibility |
| --- | --- |
| `components/app` | Shell-level UI such as sidebar, task toast, terminal dock, and migration overlay. |
| `components/common` | Shared UI such as confirmation modals. |
| `components/sdk` | SDK manager page, detail views, version cards, removal modals, and SDK list views. |
| `components/plugin` | Plugin marketplace page, plugin list/detail views, and plugin icons. |
| `components/settings` | Appearance settings, download path settings, and migration confirmation. |
| `components/sync` | SDK environment export/import view. |

### State and API Direction

The frontend dependency direction is:

```text
components -> composables -> services -> frontend/wailsjs
```

Components must not import `frontend/wailsjs` directly. Services own generated Wails imports. Composables own state and workflows. Components render the state and emit user intent.

## Events and Long-Running Tasks

Long-running vfox commands use progress events. Backend progress handling lives in `vfox_progress.go`; frontend terminal and toast state is handled by composables such as `useTaskTerminal.ts`, `useTaskToast.ts`, and related task helpers.

Migration progress uses a dedicated event path from `migration_progress.go` to app-level overlay components. The confirmation modal is rendered as a floating interface window, matching other modal behavior in the app.

## Data Locations

vfoxG manages several local data groups:

| Data | Owner |
| --- | --- |
| vfox home / download path | `config_vfox_home.go`, `config_download_path.go` |
| Custom SDK registry | `sdk_custom_registry.go` |
| Plugin description cache | `plugin_description_cache.go` |
| System SDK cache | `system_cache.go` |
| Windows path override metadata | `windows_override_metadata.go` |
| Managed SDK entrypoints | `sdk_use.go`, `sdk_custom_use.go`, platform junction/symlink helpers |

These files must stay separated. Registry code should not create links, parser code should not read files, and platform adapters should not own UI text.

## Platform Boundaries

Windows-specific behavior is isolated in `windows_*.go` files. Important responsibilities include hidden PowerShell execution, elevation, junctions, command shims, App Execution Alias compatibility, and PATH restore metadata.

Unix-specific behavior is isolated in `unix_*.go` files. It uses executable checks, symlink entrypoints, and shell profile path blocks.

Cross-platform code should call the platform-level helper names and avoid direct OS branching unless there is no existing helper.

## Testing Strategy

- Parser behavior uses table-driven tests.
- File-system code uses `t.TempDir()`.
- Environment changes use `t.Setenv()`.
- Platform-specific behavior stays in platform-specific files and tests.
- Frontend behavior is verified through `npm --prefix frontend run build`.
- Full backend verification is `go test ./...`.
