# vfoxN

<p align="center">
  <img src="build/appicon.png" alt="vfoxN" width="128" />
</p>

<p align="center">
  <strong>vfox 图形化管理界面 &middot; Wails + Vue 3</strong>
</p>

<p align="center">
  统一管理 vfox SDK 和系统自定义 SDK，一键切换，告别命令行
</p>

---

## 功能特性

### SDK 管理
- **版本管理** — 查看、安装、卸载 vfox 管理的 SDK 版本
- **一键切换** — 点击应用即可全局切换 SDK 版本，无需手动输入命令
- **自定义 SDK** — 支持将系统已安装的 SDK（如 `C:\Python314`）纳入统一管理
- **自动扫描** — 启动时自动检测系统中已安装的常见 SDK

### 插件市场
- **在线浏览** — 查看所有可用的 vfox 插件，一键添加
- **版本搜索** — 搜索远程可安装的 SDK 版本，带实时进度

### 系统集成
- **PATH 接管** — 解决 Windows 11 应用别名（App Alias）与 SDK 冲突的问题
- **Junction 架构** — 通过统一的 `~/.vfox/sdks/` 软链接管理所有 SDK，切换版本无需修改系统 PATH
- **UAC 提权** — 需要修改系统级 PATH 时自动请求管理员权限

### 其他
- **中英双语** — 支持中文和英文界面，自动跟随系统语言
- **现代 UI** — Material Design 3 风格，支持动画过渡和毛玻璃效果

## 截图

> 待补充

## 技术架构

```
+------------------------------------------+
|           Frontend (Vue 3)               |
|  SdkManager.vue | PluginMarket | Settings |
+------------------------------------------+
|        Wails v2 Bridge (RPC)             |
+------------------------------------------+
|           Backend (Go)                   |
|  app.go — SDK 管理、版本切换、PATH 操作   |
+------------------------------------------+
|         vfox CLI (core/)                 |
|  vfox.exe — 底层版本管理引擎              |
+------------------------------------------+
```

**核心设计：统一 Junction 架构**

无论是 vfox 安装的 SDK 还是用户自定义的系统 SDK，都通过 `~/.vfox/sdks/{name}` 这个 Junction（软链接）统一对外暴露。切换版本时只需更新 Junction 指向，系统 PATH 始终不变。

## 前置要求

| 工具 | 版本 | 用途 |
|------|------|------|
| [Go](https://go.dev/) | 1.21+ | 编译后端 |
| [Node.js](https://nodejs.org/) | 18+ | 编译前端 |
| [Wails CLI](https://wails.io/docs/gettingstarted/installation) | v2 | 构建框架 |
| [NSIS](https://nsis.sourceforge.io/) | 3.x | 打包安装程序（可选） |

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/yourname/vfoxN.git
cd vfoxN
```

### 2. 安装 vfox 核心

本项目运行时依赖 vfox 命令行工具，需要手动放置到 `core/` 目录：

```
vfoxN/
├── core/
│   └── vfox.exe    <- 从 https://github.com/version-fox/vfox/releases 下载
├── app.go
├── main.go
├── frontend/
└── ...
```

> `core/` 目录已被 `.gitignore` 排除，不会被提交到仓库。

### 3. 开发模式

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 启动开发模式（Go 后端 + Vite 热重载）
wails dev
```

应用窗口会自动打开。修改前端代码会自动热重载，修改 Go 代码会自动重新编译。

### 4. 构建发布版

```bash
# 构建单文件可执行程序
wails build -clean

# 构建 Windows NSIS 安装包
wails build -nsis -clean
```

构建产物位于 `build/bin/` 目录。

## 项目结构

```
vfoxN/
├── app.go                  # Go 后端核心逻辑（SDK 管理、PATH 操作）
├── main.go                 # Wails 应用入口
├── app_test.go             # 单元测试
├── app_integration_test.go # 集成测试（需 vfox 环境）
├── go.mod / go.sum         # Go 依赖
├── wails.json              # Wails 项目配置
├── core/                   # vfox 运行时（不纳入版本控制）
│   └── vfox.exe
├── frontend/               # Vue 3 前端
│   ├── src/
│   │   ├── App.vue         # 主布局（侧边栏 + 路由）
│   │   ├── components/
│   │   │   ├── SdkManager.vue    # SDK 管理主页面
│   │   │   ├── PluginMarket.vue  # 插件市场
│   │   │   └── Settings.vue      # 设置页面
│   │   ├── i18n.ts         # 国际化（中/英）
│   │   └── style.css       # 全局样式
│   └── index.html
└── build/
    └── windows/            # Windows 构建资源（图标、清单）
```

## 关键实现细节

### vfox 命令调用

所有 vfox 命令通过 `RunVfoxCommand` 统一调用，使用 `__VFOX_SHELL=cmd` 环境变量防止 vfox 弹出交互式 Shell 导致进程死锁。

### 版本切换流程

1. 用户点击应用 -> 前端乐观更新 UI
2. 后端异步执行 `vfox use --global name@version`
3. vfox 更新 `.vfox.toml` + 创建 Junction + 写入注册表
4. 后端发送 `sdk-list-changed` 事件
5. 前端刷新状态

### 系统 PATH 接管

Windows 11 的应用执行别名（如 `python.exe` -> Microsoft Store）会覆盖 vfox 的 PATH 设置。添加到系统 PATH 功能通过管理员权限将 `~/.vfox/sdks/{name}` 注入到 Machine PATH 最前面，确保 vfox 管理的版本永远优先。

## 许可证

[Apache License 2.0](LICENSE)
