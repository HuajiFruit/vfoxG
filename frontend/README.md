# vfoxG Frontend

[中文](README.zh-CN.md)

The frontend is a Vue 3 + TypeScript application embedded in Wails. It provides the GUI for SDK management, plugin marketplace actions, settings, terminal task feedback, migration progress, and SDK environment sync.

## Source Layout

```text
frontend/src/
  App.vue                       Root app composition
  app/
    navigation.ts               Navigation tab definitions
  components/
    app/                        App shell, sidebar, terminal dock, task toast
    common/                     Shared modal and common UI
    plugin/                     Plugin marketplace views
    sdk/                        SDK manager views and modals
    settings/                   Appearance and download path settings
    sync/                       SDK environment sync view
  composables/                  Vue state and workflow hooks
  services/                     Wails API wrappers
  i18n/                         English and Chinese resources
  styles/                       CSS tokens, layout, views, and components
  wailsjs/                      Generated Wails bindings
```

## Dependency Direction

```text
components -> composables -> services -> wailsjs
```

- Components render state, receive props, and emit user actions.
- Composables own refs, computed values, loading states, and workflows.
- Services call generated Wails bindings.
- User-facing text belongs in `src/i18n/`.
- Shared styles belong in `src/styles/`.

Components should not import `frontend/wailsjs` directly.

## Commands

Install dependencies:

```bash
npm install
```

Run the Vite dev server directly:

```bash
npm run dev
```

For normal desktop development, run Wails from the repository root instead:

```bash
wails dev
```

Build and type-check:

```bash
npm run build
```

From the repository root, the same build is:

```bash
npm --prefix frontend run build
```

## i18n

The app supports English and Chinese resources:

```text
src/i18n/
  en.ts
  zh.ts
  keys.ts
  index.ts
```

When adding user-facing text:

- Add the key to both `en.ts` and `zh.ts`.
- Keep key names descriptive and grouped by feature.
- Avoid hard-coded UI strings inside components.
- Keep `keys.ts` aligned with the resource shape.

## Styling

CSS is split by responsibility:

| File pattern | Purpose |
| --- | --- |
| `tokens.css` | Design tokens and common variables. |
| `base.css` | Base document and app styles. |
| `primitives.css` | Shared primitive UI styles. |
| `views.css` | Page-level view structure. |
| `sdk-*.css` | SDK manager specific styles. |
| `modals-tooltips.css` | Floating windows, modals, and tooltip behavior. |
| `responsive.css` | Responsive adjustments. |

Keep page components from accumulating large local style blocks; prefer the existing shared style files.

## Wails Bindings

Generated bindings live in `frontend/wailsjs/`. They should be changed by Wails generation, not by hand. Services in `src/services/` wrap these generated APIs so components remain decoupled from backend method names.
