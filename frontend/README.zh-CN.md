# vfoxG 前端

[English](README.md)

前端是嵌入 Wails 的 Vue 3 + TypeScript 应用。它提供 SDK 管理、插件市场、设置、终端任务反馈、迁移进度和 SDK 环境同步的图形界面。

## 源码结构

```text
frontend/src/
  App.vue                       根应用编排
  app/
    navigation.ts               导航 tab 定义
  components/
    app/                        应用壳层、侧边栏、终端停靠栏、任务提示
    common/                     通用弹窗和通用 UI
    plugin/                     插件市场视图
    sdk/                        SDK 管理视图和弹窗
    settings/                   外观和下载目录设置
    sync/                       SDK 环境同步视图
  composables/                  Vue 状态和流程 hooks
  services/                     Wails API 包装
  i18n/                         中文和英文资源
  styles/                       CSS tokens、布局、视图、组件样式
  wailsjs/                      Wails 生成绑定
```

## 依赖方向

```text
components -> composables -> services -> wailsjs
```

- 组件负责渲染状态、接收 props、发出用户动作。
- composable 负责 ref、computed、loading 状态和业务流程。
- service 调用 Wails 生成绑定。
- 用户可见文案放在 `src/i18n/`。
- 共享样式放在 `src/styles/`。

组件不应直接导入 `frontend/wailsjs`。

## 命令

安装依赖：

```bash
npm install
```

直接运行 Vite dev server：

```bash
npm run dev
```

正常桌面开发建议从仓库根目录运行 Wails：

```bash
wails dev
```

构建并类型检查：

```bash
npm run build
```

从仓库根目录执行同一个构建：

```bash
npm --prefix frontend run build
```

## i18n

应用支持中文和英文资源：

```text
src/i18n/
  en.ts
  zh.ts
  keys.ts
  index.ts
```

新增用户可见文案时：

- 同时把 key 加到 `en.ts` 和 `zh.ts`。
- key 名按功能分组，并表达清楚业务含义。
- 避免在组件里硬编码 UI 文案。
- 保持 `keys.ts` 和资源结构对齐。

## 样式

CSS 按职责拆分：

| 文件模式 | 用途 |
| --- | --- |
| `tokens.css` | 设计 token 和通用变量。 |
| `base.css` | 文档和应用基础样式。 |
| `primitives.css` | 通用基础 UI 样式。 |
| `views.css` | 页面级视图结构。 |
| `sdk-*.css` | SDK 管理专属样式。 |
| `modals-tooltips.css` | 悬浮窗口、弹窗、tooltip 行为。 |
| `responsive.css` | 响应式调整。 |

不要让页面组件继续堆积大量本地 style block，优先使用已有共享样式文件。

## Wails 绑定

生成绑定位于 `frontend/wailsjs/`。它们应由 Wails 生成，不要手写修改。`src/services/` 里的 service 负责包装这些生成 API，让组件不直接依赖后端方法名。
