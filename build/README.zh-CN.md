# 构建资源

[English](README.md)

本目录保存 vfoxG 的 Wails 构建资源、平台 manifest 和 Windows 安装脚本。

## 目录结构

```text
build/
  appicon.png                   Wails 使用的源图标
  screenshot.png                可选截图资源
  darwin/
    Info.plist                  macOS 生产 plist
    Info.dev.plist              macOS 开发 plist
  windows/
    icon.ico                    Windows 应用图标
    info.json                   Windows 版本/资源元数据
    wails.exe.manifest          Windows 应用 manifest
    installer/
      project.nsi               主 Windows 安装脚本
      project_386.nsi           32 位 Windows 安装脚本
      cleanup_vfoxg.ps1         卸载清理 helper
      wails_tools.nsh           Wails/NSIS helper 宏
```

## Windows 安装包

已验证的发布流水线会构建 Windows 安装包。amd64 安装包由 Wails 开启 NSIS 构建：

```bash
wails build -platform windows/amd64 -nsis -clean
```

386 安装包由发布流水线通过 `project_386.nsi` 直接调用 NSIS 构建。

安装器清理行为很重要，因为 vfoxG 会创建托管 SDK 入口、Windows shim 和 PATH override 元数据。修改安装器文件时，需要验证真实安装和卸载流程。

## 隐藏辅助窗口

卸载和清理 helper 不应显示可见 PowerShell 窗口。如果安装脚本需要启动 PowerShell，使用现有隐藏窗口方式，并用真实安装包确认。

## vfox Core

Release 构建会在打包前把 vfox core 下载到 `core/`。`core/` 不在 `build/` 目录下，并且已被 Git 忽略。

不要提交下载的 vfox 二进制。

## 安全编辑规则

- 除非产品行为需要，不要随意修改 Wails 生成的默认文件。
- 安装器行为变化需要同步更新 `docs/RELEASE.md` 和 `docs/RELEASE.zh-CN.md`。
- 修改 `build/windows/installer/` 时，测试安装和卸载。
- 避免把图标或 manifest 变化和无关代码行为混在一起。
