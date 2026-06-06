# 发布指南

[English](RELEASE.md)

vfoxG 在推送匹配 `v*` 的 tag 时，由 GitHub Actions 创建发布。

## 当前发布范围

已验证的工作流会构建 Windows 安装包：

| 平台 | 产物 |
| --- | --- |
| Windows amd64 | `vfoxG-windows-amd64-installer.exe` |
| Windows 386 | `vfoxG-windows-386-installer.exe` |

工作流运行在 `windows-2025`，安装 Go、Node.js、NSIS 和 Wails，下载 vfox core，然后构建并上传安装包。

## vfox Core 版本

内置 vfox 版本由 `.github/workflows/release.yml` 中的 `VFOX_VERSION` 环境变量控制。

更新前需要确认：

- 上游 vfox release 已存在。
- 两个 Windows archive 都存在：
  - `vfox_<version>_windows_x86_64.zip`
  - `vfox_<version>_windows_i386.zip`
- 尽量完成本地冒烟测试。
- Release notes 里说明 core 版本变化。

## 发布前检查清单

打 tag 前运行：

```bash
go test ./...
npm --prefix frontend run build
git diff --check
```

手动检查：

- 应用启动正常，安装/卸载辅助操作不会弹出可见控制台窗口。
- SDK 列表、详情、版本搜索、安装、卸载、使用、取消使用正常。
- 没有已发布版本的 SDK 显示可重试的“无发布版本”状态，不锁死操作按钮。
- 自定义 SDK 添加、检测、使用、移除正常。
- 插件添加和移除按用户选择保留或删除自定义 SDK 路径。
- 下载目录迁移确认窗口像其它悬浮窗口一样居中显示。
- 卸载会清理安装器设计范围内的 app data、shim、PATH/override 残留。
- 中文和英文 UI 文案显示正常。

## 打 Tag

创建并推送 annotated tag：

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

GitHub Actions 会创建或更新对应 GitHub Release，并上传 `release_staging/` 中的文件。

## 安装器资源

Windows 安装器文件位于 `build/windows/installer/`：

| 文件 | 用途 |
| --- | --- |
| `project.nsi` | 主 Windows 安装脚本。 |
| `project_386.nsi` | 32 位安装脚本。 |
| `cleanup_vfoxg.ps1` | 卸载逻辑使用的清理 helper。 |
| `wails_tools.nsh` | 安装器 helper 宏和 Wails 集成。 |

卸载时不应出现可见 PowerShell 窗口。清理行为变化时，需要用真实安装包验证，不能只从源码运行。

## 本地构建命令

通用本地构建：

```bash
wails build -clean
```

Windows amd64 安装包：

```bash
wails build -platform windows/amd64 -nsis -clean
```

Windows 386 安装包遵循发布工作流，因为它会直接调用 NSIS 和 `project_386.nsi`。

## Release Notes

Release notes 应包含：

- 用户可见的修复和功能。
- SDK、插件、PATH、迁移行为变化。
- vfox core 版本变化。
- 已知平台限制。
- 清理行为变化时的升级或卸载说明。

除非内部重构影响可维护性、测试或后续协作流程，否则不需要列入发布说明。
