# 架构

[English](ARCHITECTURE.md)

vfoxG 是一个 Wails 桌面应用。前端使用 Vue 3 和 TypeScript，后端使用 Go。后端把版本管理交给 vfox CLI core，并在它之上提供图形界面需要的状态管理、校验、迁移和平台集成。

## 运行时分层

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

每一层只负责一类事情：

| 层 | 职责 |
| --- | --- |
| Components | 渲染 UI、接收 props、发出用户动作。 |
| Composables | 管理 Vue 状态、loading、任务编排和页面流程状态。 |
| Services | 封装 Wails 生成绑定，统一前端 API 调用。 |
| App facade | 暴露稳定的 Wails 方法，并委托给后端细粒度函数。 |
| Use cases | 实现 SDK、插件、同步、迁移、配置和 PATH 流程。 |
| Parsers | 把 vfox 命令输出转换成类型化数据，不产生副作用。 |
| Platform adapters | 隔离 Windows 和 Unix 行为。 |

## 后端形态

后端当前保持在 `package main` 中。这样可以保持 Wails 绑定生成简单，同时通过原子文件控制复杂度。

### Facade 文件

`app_facade_*.go` 是前端调用的公开 API 面：

| 文件 | 公开区域 |
| --- | --- |
| `app_facade_sdk.go` | SDK 列表、详情、版本安装/使用/取消使用、自定义 SDK。 |
| `app_facade_plugin.go` | 插件市场、已添加插件、插件移除。 |
| `app_facade_path.go` | PATH 集成、override 检查、hijack/restore 操作。 |
| `app_facade_settings.go` | 下载目录、迁移选择、平台信息。 |
| `app_facade_sync.go` | SDK 环境导出、预览和导入。 |
| `app_facade_system.go` | 系统 SDK 缓存和主动扫描。 |
| `app_facade_vfox.go` | 原始 vfox 命令和带进度命令桥接。 |

facade 函数应保持很薄：校验输入、必要时获取任务锁、调用细粒度实现、发出必要事件。

### 领域和流程文件

| 模式 | 职责 |
| --- | --- |
| `model_*.go` | 与 Wails 绑定共享的 DTO 和模型结构。 |
| `config_*.go` | App config、下载目录、VFOX_HOME、JSON 持久化。 |
| `vfox_*.go` | vfox 可执行文件查找、命令执行、干净环境、进度输出、任务锁。 |
| `parse_*.go` | 已安装 SDK、详情、当前版本、搜索输出、版本归一化等纯解析器。 |
| `sdk_*.go` | SDK 清单、详情、安装、卸载、使用、运行根目录、自定义 SDK 注册/使用/检测。 |
| `plugin_*.go` | 插件市场加载、已添加状态、插件添加/移除、描述缓存。 |
| `system_*.go` | 系统 SDK 定义、扫描编排、版本探测、缓存。 |
| `path_*.go` | 跨平台路径比较和托管根目录 helper。 |
| `migration_*.go` | 下载目录迁移、非覆盖复制、进度、修复。 |
| `sync_*.go` | SDK 环境导出/导入的收集、解析、格式化和应用。 |
| `windows_*.go` | Windows PATH、shim、junction、提权 helper、override 元数据。 |
| `unix_*.go` | Unix PATH profile block、symlink、可执行检查、override 行为。 |

## 前端形态

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

### 组件分组

| 目录 | 职责 |
| --- | --- |
| `components/app` | 应用壳层 UI，例如侧边栏、任务提示、终端停靠栏、迁移遮罩。 |
| `components/common` | 通用 UI，例如确认弹窗。 |
| `components/sdk` | SDK 管理页面、详情视图、版本卡片、移除弹窗、SDK 列表视图。 |
| `components/plugin` | 插件市场页面、插件列表/详情视图、插件图标。 |
| `components/settings` | 外观设置、下载目录设置、迁移确认。 |
| `components/sync` | SDK 环境导出/导入视图。 |

### 状态和 API 方向

前端依赖方向是：

```text
components -> composables -> services -> frontend/wailsjs
```

组件不直接导入 `frontend/wailsjs`。service 负责导入生成绑定，composable 负责状态和流程，组件负责渲染状态并表达用户意图。

## 事件和长任务

长时间运行的 vfox 命令使用进度事件。后端进度处理在 `vfox_progress.go`，前端终端和任务提示状态由 `useTaskTerminal.ts`、`useTaskToast.ts` 等 composable 处理。

迁移进度通过 `migration_progress.go` 到应用级遮罩组件。迁移确认弹窗渲染为悬浮在界面上的窗口，与应用里其它 modal 行为保持一致。

## 数据位置

vfoxG 管理几类本地数据：

| 数据 | 负责文件 |
| --- | --- |
| vfox home / 下载目录 | `config_vfox_home.go`、`config_download_path.go` |
| 自定义 SDK 注册表 | `sdk_custom_registry.go` |
| 插件描述缓存 | `plugin_description_cache.go` |
| 系统 SDK 缓存 | `system_cache.go` |
| Windows PATH override 元数据 | `windows_override_metadata.go` |
| 托管 SDK 入口 | `sdk_use.go`、`sdk_custom_use.go`、平台 junction/symlink helper |

这些文件必须保持职责分离。registry 代码不创建链接，parser 代码不读文件，平台适配层不持有 UI 文案。

## 平台边界

Windows 专属行为隔离在 `windows_*.go`。重点包括隐藏 PowerShell、提权、junction、命令 shim、应用执行别名兼容和 PATH restore 元数据。

Unix 专属行为隔离在 `unix_*.go`。它使用可执行检查、symlink 入口和 shell profile path block。

跨平台代码应调用已有平台 helper，除非没有合适的 helper，否则不要直接写 OS 分支。

## 测试策略

- parser 使用 table-driven tests。
- 文件系统逻辑使用 `t.TempDir()`。
- 环境变量变更使用 `t.Setenv()`。
- 平台行为放进平台文件和平台测试。
- 前端行为通过 `npm --prefix frontend run build` 验证。
- 后端完整验证命令是 `go test ./...`。
