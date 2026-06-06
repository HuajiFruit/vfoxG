# 开发指南

[English](DEVELOPMENT.md)

本文说明如何在本地配置、运行、验证和修改 vfoxG。

## 环境要求

| 工具 | 版本 | 说明 |
| --- | --- | --- |
| Go | 1.23+ | `go.mod` 声明 Go 1.23.0。 |
| Node.js | 22+ | 与发布流水线保持一致。 |
| npm | 随 Node 提供 | 用于 Vite 前端。 |
| Wails CLI | v2 | 用于 `wails dev` 和 `wails build`。 |
| NSIS | 3.x | 只有构建 Windows 安装包时需要。 |

如果本机没有 Wails：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 初次配置

从仓库根目录安装前端依赖：

```bash
npm --prefix frontend install
```

本地运行时需要把 vfox core 放到 `core/`：

```text
core/
  windows/
    x86_64/vfox.exe
    x86/vfox.exe
```

Release 构建会自动下载这个目录，本地开发需要手动提供。`core/` 已被 Git 忽略。

## 本地运行

启动 Wails 开发模式：

```bash
wails dev
```

Wails 会同时启动 Go 后端和 Vite 前端。前端 dev server 命令在 `wails.json` 中配置。

## 验证命令

运行后端测试：

```bash
go test ./...
```

构建并类型检查前端：

```bash
npm --prefix frontend run build
```

检查 Git 变更里的空白问题：

```bash
git diff --check
```

Go-only 变更在测试前对改动文件运行 `gofmt`。前端变更需要通过前端 build 命令保持 TypeScript 和 Vue 编译正常。

## Wails 绑定

生成的前端绑定在 `frontend/wailsjs/`。它们来自 Go 的导出方法和模型结构。

修改导出的 `App` 方法或 DTO 时：

- 公开方法放在匹配的 `internal/app/app_facade_*.go` 文件里。
- 导出的 DTO 放在 `internal/model/`。
- 必要时重新构建或运行应用，让 Wails 重新生成绑定。
- 只有 API 面真正变化时，才提交生成绑定的变化。

## 后端开发流程

后端现在拆在 `internal/` 下：

1. `main.go` 只负责 Wails 启动和绑定入口。
2. 公开 Wails 方法放在 `internal/app/app_facade_*.go`。
3. 有状态业务流程放进细粒度 `internal/app/*` 文件，例如 `sdk_use.go`、`plugin_remove.go`、`sync_import_apply.go`。
4. 导出的 DTO 放在 `internal/model/`。
5. 纯 vfox 输出解析器放在 `internal/parser/`，并用 table-driven tests 覆盖。
6. Windows 行为放在 `internal/app/windows_*.go`，Unix 行为放在 `internal/app/unix_*.go`。

不要把后端业务逻辑放回仓库根目录。不要继续往 `internal/app/app.go` 里塞新流程，它应主要负责 App 状态。

## 前端开发流程

前端源码位于 `frontend/src/`：

1. 组件负责渲染状态和发出用户事件。
2. composable 负责 UI 状态和异步流程。
3. service 调用 Wails 生成绑定。
4. 文案放进 `frontend/src/i18n/`。
5. 共享样式放进 `frontend/src/styles/`。

新增用户可见文案时，同时更新 `frontend/src/i18n/en.ts` 和 `frontend/src/i18n/zh.ts`，并通过 `frontend/src/i18n/keys.ts` 保持 key 对齐。

## 调试建议

- 长时间 vfox 操作先看终端停靠栏和任务提示输出。
- 如果 vfox CLI 能切换 SDK 但应用不能，检查托管入口路径和 PATH override 状态。
- Windows 上如果命令打开了错误的可执行文件，检查应用执行别名冲突。
- 迁移异常时，先检查迁移进度事件和新旧 VFOX_HOME，再改 repair 逻辑。
- 卸载后有残留时，先检查 Windows shim 文件和 override 元数据，再改安装器清理。

## 文档更新

行为变化时，同一个变更里同步更新文档：

| 变化 | 需要检查的文档 |
| --- | --- |
| 新后端模块或文件模式 | `docs/ARCHITECTURE.md`、`docs/ARCHITECTURE.zh-CN.md` |
| 新开发命令 | `docs/DEVELOPMENT.md`、`docs/DEVELOPMENT.zh-CN.md` |
| 发布流程变化 | `docs/RELEASE.md`、`docs/RELEASE.zh-CN.md` |
| 命名、注释、拆分规则 | `docs/CODE_STYLE.en.md`、`docs/CODE_STYLE.md` |
| 用户可见功能 | 根 `README.md` 和 `READMEcn.md` |
