# vfoxG

[English README](README.md)

<p align="center">
  <img src="build/appicon.png" alt="vfoxG" width="128" />
</p>

<p align="center">
  <strong>vfox 图形化管理界面 · Wails + Vue 3</strong>
</p>

<p align="center">
  统一管理 vfox SDK 和系统自定义 SDK，在图形界面中完成安装、卸载和版本切换。
</p>

---

## 截图

![vfoxG 截图](build/screenshot.png)

## 下载

最新版本可以在 [GitHub Releases](https://github.com/HuajiFruit/vfoxG/releases) 下载。

vfoxG v0.3.0 通过 GitHub Actions 构建以下平台产物：

| 平台 | 产物 |
| --- | --- |
| Windows amd64 | `vfoxG-windows-amd64-installer.exe` |
| Windows 386 | `vfoxG-windows-386-installer.exe` |
| Linux amd64 | portable `.tar.gz`、`.deb`、`.rpm` |
| macOS Apple Silicon | `vfoxG-macos-arm64.dmg` |

## 功能

### SDK 管理

- 查看、安装、卸载和切换 vfox 管理的 SDK 版本。
- 将系统中已经安装的 SDK 添加为自定义 SDK。
- 自动扫描常见系统 SDK。
- 通过 `~/.vfox/sdks/{name}` 链接统一暴露 SDK 路径。

### 插件市场

- 浏览可用的 vfox 插件。
- 在图形界面中添加插件。
- 搜索可安装的 SDK 版本，并显示进度反馈。

### 系统集成

- 在应用中管理 PATH 集成。
- 处理 Windows 应用执行别名与 SDK 命令冲突的问题。
- 仅在需要系统级变更时请求管理员权限。
- Linux 和 macOS 使用各自的 shell profile 处理逻辑。

## 架构

```text
+------------------------------------------+
|           Frontend (Vue 3)               |
|  SdkManager.vue | PluginMarket | Settings |
+------------------------------------------+
|        Wails v2 Bridge (RPC)             |
+------------------------------------------+
|           Backend (Go)                   |
|  app.go - SDK 管理、版本切换、PATH 处理   |
+------------------------------------------+
|         vfox CLI (core/)                 |
|  发布包内置或本地手动提供的运行时         |
+------------------------------------------+
```

## vfox Core

Release 安装包会内置对应平台的 vfox core。

本地开发时，需要把 vfox 放到对应的 `core/` 目录：

```text
core/
  windows/
    x86_64/vfox.exe
    x86/vfox.exe
  linux/
    x86_64/vfox
  macos/
    arm64/vfox
```

`core/` 目录已被 Git 忽略，不会提交到仓库。

## 开发

### 环境要求

| 工具 | 版本 |
| --- | --- |
| Go | 1.23+ |
| Node.js | 22+ |
| Wails CLI | v2 |
| NSIS | 3.x，仅 Windows 安装包需要 |
| nfpm | 仅 Linux deb/rpm 打包需要 |

### 运行

```bash
cd frontend
npm install
cd ..
wails dev
```

### 测试

```bash
go test ./...
npm --prefix frontend run build
```

### 本地构建

```bash
wails build -clean
```

Windows 安装包：

```bash
wails build -platform windows/amd64 -nsis -clean
```

## 发布流程

推送 `v*` tag 后，GitHub Actions 会自动构建并上传 Release 产物。

```bash
git tag -a v0.3.0 -m "v0.3.0"
git push origin v0.3.0
```

发布工作流会下载 vfox core、构建各平台包，并自动上传到 GitHub Release。

## 项目结构

```text
vfoxG/
  app.go                  Go 后端核心逻辑
  app_windows.go          Windows 平台集成
  app_unix.go             Linux/macOS 共用集成逻辑
  app_linux.go            Linux shell profile 集成
  app_darwin.go           macOS shell profile 集成
  frontend/               Vue 3 前端
  build/windows/          Windows 安装包资源
  .github/workflows/      GitHub Actions 发布流水线
  nfpm.yaml.tmpl          Linux deb/rpm 打包模板
  wails.json              Wails 项目配置
```

## 致谢

特别感谢 [vfox](https://github.com/version-fox/vfox) 项目提供跨平台版本管理引擎，vfoxG 基于它的能力构建图形化体验。

vfoxG 是独立的第三方图形化界面，与 vfox 项目无官方关联，也非其官方产品。
