# vfoxG 代码规范与细粒度拆分方案

[English](CODE_STYLE.en.md)

本文档是 vfoxG 的可维护性约束和后续重构蓝图。重点不是“把文件换个地方”，而是把职责拆到足够小，让每个文件都能独立理解、独立测试、独立修改。

当前阶段先按本文档作为目标规范；真正拆代码时必须小步迁移，每一批迁移后都要能编译、能测试、能回滚。

## 可维护性目标

- 一个文件只承载一个原子职责。
- 一个函数只完成一个行为步骤。
- 一个模块只暴露少量稳定入口。
- 状态变更必须可测试。
- 平台差异必须隔离。
- 前端不直接处理系统状态，后端不处理 UI 展示细节。
- 新功能不得继续塞进已经过大的文件。

建议阈值：

| 类型 | 建议上限 | 超过后的处理 |
| --- | ---: | --- |
| Go 普通文件 | 250 行 | 拆到更小原子文件 |
| Go 平台文件 | 300 行 | 按 PATH、进程、文件系统、提权拆分 |
| Go 函数 | 60 行 | 拆成 validate/plan/apply/parse 等 helper |
| Vue 页面组件 | 300 行 | 拆子组件或 composable |
| Vue 子组件 | 220 行 | 拆 presentation 与 behavior |
| composable/service | 180 行 | 按状态域继续拆 |
| CSS 文件 | 400 行 | 拆 tokens/layout/components/views |

这些不是硬性编译规则，但 code review 时应以它们作为拆分判断标准。

## 依赖方向

后端依赖方向：

```text
app facade
  -> usecase/service
    -> domain model/parser
    -> platform adapter
    -> storage
```

禁止反向依赖：

- parser 不依赖 App。
- model 不依赖 runtime/Wails。
- platform adapter 不依赖 UI event。
- storage 不调用 vfox command。
- command executor 不解析具体业务结果。

前端依赖方向：

```text
views/components
  -> composables
    -> services
      -> wailsjs generated bindings
```

禁止：

- 组件里直接堆大量 Wails import。
- service 反向依赖组件状态。
- composable 操作 DOM 或写样式。
- i18n 文案散落在组件里。

## 优化后的目录结构

当前结构已经进入 `internal/` 分层，根目录只保留 `main.go` 作为 Wails 启动入口。后端新代码不得继续加到仓库根目录。

```text
vfoxG/
  main.go                        Wails 启动和 app 绑定入口
  internal/
    app/                         Wails facade 和有状态后端流程
    model/                       与生成前端绑定共享的 DTO
    parser/                      vfox 命令输出纯解析器
    pathutil/                    路径比较和 PATH 清洗 helper
    storage/                     JSON 文件持久化 helper
  frontend/                      Vue 3 前端
  build/                         图标、manifest、安装脚本
  docs/                          中英双语项目文档
```

后续如果继续拆包，优先从 `internal/app` 中拆出真正有独立边界的领域包，例如 `sdk`、`plugin`、`pathenv`、`migration`。拆包必须伴随测试和 Wails 绑定验证。

### 后端目录职责

| 目录 | 只负责 |
| --- | --- |
| `main.go` | Wails 启动、窗口生命周期、绑定 `internal/app.App`。 |
| `internal/app/` | 有状态业务流程、Wails facade、配置读写、vfox 命令、平台适配、迁移、同步。 |
| `internal/model/` | 导出的 DTO 和 Wails 生成绑定需要的模型。 |
| `internal/parser/` | vfox 命令输出纯解析器，不读文件、不发事件、不调用 vfox。 |
| `internal/pathutil/` | 路径比较、PATH 清洗等无应用状态的 helper。 |
| `internal/storage/` | JSON 文件持久化等无应用状态的 helper。 |
## 后端原子文件拆分细则

### App facade

`app_facade_*.go` 只允许做：

- trim/validate 入参。
- 获取任务锁。
- 调用 usecase。
- emit 必要事件。

禁止：

- 写文件。
- 执行 vfox 命令。
- 解析 vfox 输出。
- 操作 PATH。
- 拼复杂路径。

### vfox command

`vfox_command.go` 只负责启动短命令：

- context timeout。
- 设置 hidden window。
- 设置 cleaned env。
- 收集 combined output。
- ANSI 清洗。

禁止：

- 根据具体 SDK 名解析版本。
- 修改文件系统。
- 修改 App 状态。

`vfox_progress.go` 只负责长任务和实时输出：

- stdout/stderr scanner。
- progress line emit。
- version not released 输出归一化。
- `[DONE]` / `[EXIT ERROR]` 事件。

禁止：

- 安装后刷新 SDK 列表。
- 写 cache。
- 修改 PATH。

### parser

parser 文件必须是纯函数：

- 输入 string。
- 输出 struct/slice/string。
- 不读文件。
- 不调用 vfox。
- 不发事件。

拆分：

| 文件 | 函数 |
| --- | --- |
| `parse_installed.go` | `parseInstalledSdksOutput` |
| `parse_sdk_detail.go` | `parseSdkDetailOutput`、`parseSdkDetailVersionLine` |
| `parse_current.go` | `parseCurrentSdkVersion` |
| `parse_search.go` | `parseSearchVersionsOutput` |
| `parse_version.go` | `normalizeSdkVersion`、`sameSdkVersion` |

每个 parser 必须有 table-driven tests。

### SDK use/install/uninstall

拆分成独立状态流：

| 文件 | 只负责 |
| --- | --- |
| `sdk_install.go` | `vfox install` 和安装后刷新事件 |
| `sdk_uninstall.go` | 卸载版本、处理当前版本被卸载 |
| `sdk_use.go` | 应用/取消应用版本 |
| `sdk_runtime_root.go` | 从 `vfox info` 路径解析真实 runtime root |
| `sdk_detail.go` | 当前版本、已安装版本、自定义版本合并前的数据 |

`sdk_use.go` 的应用流程必须显式分步：

```text
validate
lock
clear active custom link if needed
run vfox use
resolve runtime root
refresh sdks/<name> entrypoint
refresh path override if enabled
emit events
```

这些步骤不允许折叠成一个大函数。

### custom SDK

自定义 SDK 分成三类文件：

| 文件 | 只负责 |
| --- | --- |
| `sdk_custom_registry.go` | JSON registry 增删改查 |
| `sdk_custom_detect.go` | 对 exe 执行 version 命令 |
| `sdk_custom_use.go` | 用 junction/symlink 激活自定义 SDK |

禁止 registry 文件创建 junction；禁止 detect 文件写 JSON。

### plugin

插件拆分：

| 文件 | 只负责 |
| --- | --- |
| `plugin_market.go` | available/refresh market |
| `plugin_state.go` | added 状态、插件目录扫描 |
| `plugin_add.go` | add 插件流程 |
| `plugin_remove.go` | remove 插件流程 |
| `plugin_cache.go` | GUI 插件缓存文件 |

`plugin_remove.go` 必须把“保留自定义 SDK”“恢复 PATH override”“删除插件数据”拆成 helper。

### system SDK scan

系统 SDK 扫描拆分：

| 文件 | 只负责 |
| --- | --- |
| `system_defs.go` | SDK 定义表 |
| `system_find.go` | PATH 查找候选 executable |
| `system_version.go` | 运行 version 命令和过滤无效输出 |
| `system_scan.go` | 并发扫描编排 |
| `system_cache.go` | GUI cache 读写 |

扫描逻辑禁止修改全局 `os.Setenv`。

### PATH / override

公共 PATH：

| 文件 | 只负责 |
| --- | --- |
| `path_common.go` | split/dedupe/clean path |
| `path_roots.go` | vfox managed roots |
| `path_compare.go` | samePath/isPathWithin |

Windows：

| 文件 | 只负责 |
| --- | --- |
| `windows_path_user.go` | User PATH 中 vfox core 增删查 |
| `windows_path_machine.go` | Machine PATH SDK override 增删查 |
| `windows_env_broadcast.go` | 环境变量变更广播 |
| `windows_shim_aliases.go` | alias 列表和 safe name |
| `windows_shim_script.go` | `.cmd` 文本生成 |
| `windows_shim_store.go` | shim 文件写入/删除 |
| `windows_override_metadata.go` | `hijacked_paths.json` 读写和旧格式兼容 |
| `windows_override_apply.go` | 启用/刷新 override |
| `windows_override_restore.go` | 关闭 override 和恢复原 PATH |

`windows_override_metadata.go` 只读写 JSON，不执行提权、不改 PATH。

Unix：

| 文件 | 只负责 |
| --- | --- |
| `unix_profile.go` | profile 路径候选 |
| `unix_path_block.go` | managed block 生成/删除 |
| `unix_override.go` | SDK PATH block 应用/恢复 |

### migration

迁移拆分：

| 文件 | 只负责 |
| --- | --- |
| `migration_detect.go` | 是否存在可迁移数据 |
| `migration_plan.go` | 生成迁移计划，列出源/目标 |
| `migration_copy_file.go` | 文件 copy no overwrite |
| `migration_copy_dir.go` | 目录 copy no overwrite |
| `migration_copy_link.go` | symlink/junction 迁移 |
| `migration_progress.go` | 进度统计和事件 |
| `migration_repair.go` | 迁移后修复 SDK link/PATH override |

迁移函数必须先 plan，再 apply。apply 中失败不能改变 app config。

### import/export

拆分：

| 文件 | 只负责 |
| --- | --- |
| `sync_export_collect.go` | 收集当前环境 |
| `sync_export_format.go` | 格式化 TXT |
| `sync_import_parse.go` | 解析 TXT |
| `sync_import_plan.go` | 生成导入计划 |
| `sync_import_apply.go` | 执行导入 |

parse 和 format 必须是纯函数。

## 前端细粒度拆分方案

### 目标结构

```text
frontend/src/
  app/
    AppShell.vue                 顶层布局
    navigation.ts                tab 顺序、切换逻辑
    events.ts                    Wails 全局事件注册

  services/
    sdkService.ts                SDK Wails API
    pluginService.ts             plugin Wails API
    settingsService.ts           settings Wails API
    syncService.ts               import/export Wails API
    platformService.ts           platform Wails API

  composables/
    useNotify.ts
    useTerminalTask.ts
    useSdkList.ts
    useSdkDetail.ts
    useSdkVersionActions.ts
    useCustomSdkActions.ts
    usePathOverride.ts
    usePluginMarket.ts
    useSettingsDownloadPath.ts
    useMigrationDialog.ts

  components/
    common/
      ConfirmModal.vue
      IconButton.vue
      LoadingSpinner.vue
      PathBox.vue
      ToastHost.vue

    sdk/
      SdkManager.vue             只做页面编排
      SdkSidebar.vue
      SdkListItem.vue
      SdkDetailPanel.vue
      SdkHeader.vue
      SdkPathOverrideActions.vue
      VersionList.vue
      VersionCard.vue
      VersionSearchPanel.vue
      CustomSdkList.vue
      CustomSdkForm.vue
      RemovePluginModal.vue

    plugin/
      PluginMarket.vue           只做页面编排
      PluginList.vue
      PluginListItem.vue
      PluginDetail.vue
      PluginVersionSearch.vue
      PluginIcon.vue

    settings/
      Settings.vue               只做页面编排
      AppearanceSettings.vue
      TerminalSettings.vue
      DownloadPathSettings.vue
      MigrationConfirmModal.vue

  i18n/
    index.ts
    en.ts
    zh.ts
    keys.ts

  styles/
    tokens.css
    base.css
    layout.css
    components.css
    sdk.css
    plugin.css
    settings.css
```

### Vue 文件职责边界

页面组件只做编排：

- 加载 composable。
- 传 props。
- 接事件。
- 不写复杂业务逻辑。

子组件只做展示：

- props 输入。
- emits 输出用户动作。
- 不直接调用 Wails。
- 不直接改全局状态。

composable 只做状态和动作：

- 持有 refs/computed。
- 调用 service。
- 处理 loading/error。
- 不写模板。

service 只做 API 包装：

- 调用 `frontend/wailsjs`。
- 统一错误类型。
- 不持有 Vue ref。

### SdkManager 拆分目标

`SdkManager.vue` 最终只保留：

```text
读取 useSdkList/useSdkDetail/useSdkVersionActions
渲染 SdkSidebar
渲染 SdkDetailPanel
处理页面级 tab 或 selection
```

必须迁出的内容：

| 逻辑 | 目标 |
| --- | --- |
| SDK 列表加载 | `useSdkList.ts` |
| SDK detail 拉取 | `useSdkDetail.ts` |
| 应用/取消应用/安装/卸载 | `useSdkVersionActions.ts` |
| 自定义 SDK 添加/删除/应用 | `useCustomSdkActions.ts` |
| PATH override 启用/停用 | `usePathOverride.ts` |
| 版本搜索 | `VersionSearchPanel.vue` + composable |
| 版本卡片 | `VersionCard.vue` |
| 路径复制显示 | `PathBox.vue` |

### PluginMarket 拆分目标

| 逻辑 | 目标 |
| --- | --- |
| 市场列表加载 | `usePluginMarket.ts` |
| 插件列表展示 | `PluginList.vue` |
| 插件详情 | `PluginDetail.vue` |
| 版本搜索和安装 | `PluginVersionSearch.vue` |
| 插件图标 | `PluginIcon.vue` |
| 插件 API 调用 | `pluginService.ts` |

### Settings 拆分目标

| 逻辑 | 目标 |
| --- | --- |
| 主题设置 | `AppearanceSettings.vue` |
| 终端显示开关 | `TerminalSettings.vue` |
| 下载目录设置 | `DownloadPathSettings.vue` |
| 迁移确认弹窗 | `MigrationConfirmModal.vue` |
| 下载目录状态 | `useSettingsDownloadPath.ts` |
| 迁移确认状态 | `useMigrationDialog.ts` |

## 变量命名规范

变量名必须先表达业务含义，再表达类型或状态。除极小局部作用域外，不允许使用 `data`、`info`、`list`、`item`、`tmp`、`obj`、`res` 这类无法说明业务语义的名称。

### 通用命名原则

| 场景 | 规范 | 示例 |
| --- | --- | --- |
| 布尔值 | 使用 `is`、`has`、`can`、`should`、`allow`、`enable` 开头 | `isInstalled`、`hasReleaseVersion`、`canUseSdk` |
| 加载状态 | 使用动作进行时，避免只有 `loading` | `loadingSdkList`、`savingSettings`、`removingPlugin` |
| 数量 | 使用 `Count` 后缀 | `versionCount`、`installedSdkCount` |
| 下标 | 使用 `Index` 后缀；`i/j` 只允许极小循环 | `selectedVersionIndex` |
| 文件路径 | 文件路径使用 `Path`，目录使用 `Dir`，根目录使用 `Root` | `sdkPath`、`downloadDir`、`vfoxRoot` |
| 版本 | 使用完整 `Version`，避免业务代码里写 `ver` | `currentVersion`、`targetVersion` |
| SDK | 明确 SDK 语义，不使用单独的 `name` 贯穿业务层 | `sdkName`、`selectedSdkName` |
| 插件 | 明确插件语义 | `pluginName`、`pluginDir` |
| 同类变量 | 禁止只用数字后缀区分；如果值完全相同，必须先删除重复变量，只保留一个按业务含义命名的变量；只有来源、用途或生命周期不同，才允许拆成多个名字 | `sdkIconImage`；确实不同才用 `sourceImage`、`previewImage` |
| 切片/数组 | 使用复数名词 | `sdks`、`versions`、`plugins` |
| Map | 使用 `By` 或 `Map` 表达索引关系 | `sdkByName`、`versionStatusMap` |
| Set | 使用 `Set` 后缀 | `installedVersionSet` |
| 错误 | 普通错误用 `err`，保留错误用业务前缀 | `err`、`installErr`、`migrationErr` |

### Go 变量命名

- 局部短变量只允许出现在很小作用域内，例如 `err`、`ok`、`i`、`j`、`wg`、`mu`、`ctx`、`cancel`。
- 跨函数或业务层传递的变量必须使用完整业务名，例如 `sdkName`、`targetVersion`、`downloadDir`。
- `sync.Mutex` / `sync.RWMutex` 使用 `Mu` 后缀，例如 `taskMu`、`cacheMu`。
- `sync.WaitGroup` 可以使用 `wg`，但只允许在单个函数内。
- `context.Context` 固定使用 `ctx`，取消函数固定使用 `cancel`。
- 包级变量必须表达生命周期和用途，避免 `cache`、`state` 这类泛名；可使用 `sdkCache`、`taskState`。
- 临时变量必须靠近使用位置声明，不能提前堆在函数顶部。

### 前端变量命名

- `ref` / `reactive` 状态使用名词或状态名，例如 `selectedSdk`、`activeTab`、`downloadDir`。
- 布尔状态必须可直接读成判断语句，例如 `isUsingSdk`、`hasReleaseVersion`、`canInstallVersion`。
- 异步状态带动作语义，例如 `loadingVersions`、`installingVersion`、`migratingDownloadDir`。
- 计算属性使用名词或布尔前缀，例如 `visibleVersions`、`isVersionSelectable`。
- 事件处理函数使用 `handleXxx`，例如 `handleSelectSdk`、`handleInstallVersion`。
- 打开确认类窗口使用 `requestXxx`，真正执行操作使用 `confirmXxx`，例如 `requestMigration`、`confirmMigration`。
- API 调用函数使用动作动词，例如 `fetchSdkDetail`、`saveDownloadDir`、`removeCustomSdk`。
- 组件 props 使用业务名词，emit 事件使用动作名，例如 prop `sdkName`，emit `select-sdk`。

### 命名示例

| 不推荐 | 推荐 | 原因 |
| --- | --- | --- |
| `data` | `sdkDetails` | 表达数据内容 |
| `list` | `installedSdks` | 表达集合内容和状态 |
| `flag` | `hasReleaseVersion` | 布尔值必须说明判断含义 |
| `path` | `sdkPath` / `downloadDir` | 路径必须说明指向对象 |
| `ver` | `targetVersion` | 业务代码里避免缩写 |
| `name` | `sdkName` / `pluginName` | 避免 SDK 和插件语义混淆 |
| `image` / `image1` | 删除 `image1` 这类重复状态，只保留一个有业务含义的变量，例如 `sdkIconImage` | 两者完全一样说明它们不是两个概念，不能只改名后继续保留两份 |
| `file` / `file2` | `configFile` / `backupFile` | 同类变量必须按用途命名 |
| `m` | `sdkByName` | Map 必须表达 key 的含义 |
| `result` | `installResult` | 返回值必须说明业务来源 |
| `show` | `isMigrationDialogOpen` | UI 状态必须说明控制对象 |

如果两个变量只能通过 `1`、`2`、`New`、`Old` 区分，必须先判断它们是不是同一份数据。像 `image` 和 `image1` 完全一样时，它们只能合并成一个变量，不能为了通过命名检查，把它们改成 `sdkIconImage` / `previewImage` 后继续保留两份。

- 数据完全相同：删除重复变量，只保留一个变量，并按业务含义命名，例如 `sdkIconImage`；所有使用位置共用这一份状态，不能保留 `image` 和 `image1` 两份相同状态。
- 来源、值、生命周期都相同：直接合并引用，不允许拆成两个 `ref`、两个 `computed`、两个 struct 字段或两份缓存。
- 只有名称不同、内容完全一样：这是重复状态问题，不是命名问题；处理方式是合并变量，而不是制造两个“看起来正确”的名称。

如果确实不是同一份数据，再按真实差异命名：

- 来源不同：使用 `localImage`、`remoteImage`、`defaultConfigFile`。
- 用途不同：使用 `previewImage`、`iconImage`、`installerFile`。
- 状态不同：使用 `currentVersion`、`targetVersion`、`previousDownloadDir`。

## 代码写法统一

### Go 函数模板

状态变更函数：

```go
func (a *App) UseVersion(name, version string) (string, error) {
    name = strings.TrimSpace(name)
    version = strings.TrimSpace(version)
    if name == "" || version == "" {
        return "", fmt.Errorf("plugin name and version cannot be empty")
    }

    releaseTask, err := a.tryStartVfoxTask()
    if err != nil {
        a.emitEvent("vfox-busy")
        return "", err
    }
    defer releaseTask()

    return a.useVersionUnlocked(name, version)
}
```

纯函数：

```go
func parseCurrentSdkVersion(name string, out string) string {
    name = strings.TrimSpace(name)
    if name == "" || strings.TrimSpace(out) == "" {
        return ""
    }
    // parse only, no side effects
}
```

文件 IO：

```go
func readAppConfig(path string) (AppConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return AppConfig{}, nil
        }
        return AppConfig{}, err
    }
    ...
}
```

### Vue 异步模板

```ts
const handleUse = async (name: string, version: string) => {
  usingVersion.value = version;
  try {
    await runTask(t('task.version.switch', { name, version }), async () => {
      await sdkService.useVersion(name, version);
      await refreshSdkState(name);
    });
  } catch (err) {
    notifyTaskError(err, t('sdk.switch_error', { name, version }));
  } finally {
    usingVersion.value = null;
  }
};
```

## 注释写法统一

### 注释只解释原因，不复述代码

不要写：

```go
// Remove file.
os.Remove(path)
```

应该写：

```go
// vfox can leave a stale junction after switching versions.
// Refresh it so terminal shims resolve the newly selected SDK.
a.removeJunctionIfExists(sdkLinkPath)
```

### 必须注释的场景

- Windows 提权和隐藏窗口。
- PATH User/Machine scope 差异。
- junction/symlink 与 vfox runtime root 的关系。
- 兼容旧 JSON/TOML/CLI 输出。
- 删除数据和卸载流程。
- 迁移失败时如何保证不污染新配置。

### 公开方法注释

公开方法用英文，并以方法名开头：

```go
// UseVersion switches the global vfox SDK version and refreshes managed entrypoints.
func (a *App) UseVersion(name, version string) (string, error) {
```

内部复杂 helper 可以用中文或英文，但同一个文件内尽量统一。

## 测试结构

测试文件跟随原子文件命名：

```text
parse_current.go              parse_current_test.go
parse_sdk_detail.go           parse_sdk_detail_test.go
sdk_runtime_root.go           sdk_runtime_root_test.go
windows_override_metadata.go  windows_override_metadata_test.go
migration_copy_file.go        migration_copy_file_test.go
```

测试规则：

- parser 全部 table-driven。
- 文件系统逻辑全部用 `t.TempDir()`。
- 环境变量用 `t.Setenv()`。
- Windows 专属测试放 `*_windows_test.go`。
- 不依赖用户真实 SDK 状态。
- 不让测试改真实 PATH。

## 拆分验收标准

每次拆分 PR/提交必须满足：

- 没有行为变化，除非提交说明中明确写出。
- `go test ./...` 通过。
- 涉及前端时 `npm --prefix frontend run build` 通过。
- 拆出的新文件职责单一。
- 旧文件行数下降。
- 新文件有对应测试或已有测试覆盖。
- Wails 绑定没有无关变更。

## 禁止事项

- 禁止继续扩大 `app.go`、`app_windows.go`、`SdkManager.vue`、`PluginMarket.vue`。
- 禁止一个新文件同时包含 parser、IO、状态变更。
- 禁止组件直接调用多个 Wails API 后再自己拼业务流程。
- 禁止在 parser 中读文件或发事件。
- 禁止在 service/composable 中写样式。
- 禁止为了拆目录一次性移动大量文件且不拆职责。
- 禁止无测试拆分 SDK、PATH、迁移、卸载逻辑。

## 推荐拆分顺序

1. 拆 parser 纯函数，风险最低。
2. 拆 model 和 DTO。
3. 拆 config/VFOX_HOME。
4. 拆 vfox command/progress/task lock。
5. 拆 SDK runtime root 和 use/unuse。
6. 拆 Windows shim/override metadata。
7. 拆 system SDK scan。
8. 拆 migration。
9. 拆 frontend service。
10. 拆 frontend composable。
11. 拆 Vue 子组件。
12. 最后再考虑 package 化。
