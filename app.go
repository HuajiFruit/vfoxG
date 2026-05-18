package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.ScanSystemSdks()
	go a.RefreshAvailablePlugins() // 后台预热插件市场缓存
}

// getCleanedEnvForVfox returns a copy of the current environment but sanitizes the PATH
// to remove any previously injected vfox sdks, ensuring `vfox` reads from .vfox.toml.
func (a *App) getCleanedEnvForVfox() []string {
	env := os.Environ()
	vfoxSdksDir := filepath.Join(a.getVfoxHome(), "sdks")
	sep := string(filepath.ListSeparator) // ";" on Windows, ":" on Unix

	for i, e := range env {
		if strings.HasPrefix(strings.ToLower(e), "path=") {
			pathVal := e[5:]
			parts := strings.Split(pathVal, sep)
			var clean []string
			for _, p := range parts {
				pTrim := strings.TrimSpace(p)
				if pTrim == "" {
					continue
				}
				if strings.HasPrefix(strings.ToLower(pTrim), strings.ToLower(vfoxSdksDir)) {
					continue
				}
				clean = append(clean, pTrim)
			}
			env[i] = "PATH=" + strings.Join(clean, sep)
			break
		}
	}
	// 添加伪装变量，使用 cmd/bash 可以避免 vfox 弹出子 shell 而导致死锁
	shellName := "bash"
	if stdruntime.GOOS == "windows" {
		shellName = "cmd"
	}
	return append(env, "__VFOX_SHELL="+shellName)
}

// RunVfoxCommand 执行短生命周期的 vfox 命令，带 15s 超时防止前端卡死
func (a *App) RunVfoxCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, filepath.Join(a.getCoreDir(), getVfoxExeName()), args...)

	hideWindow(cmd)
	// 伪装为 cmd 以防止 vfox 尝试重新加载父进程（导致应用多开的严重 BUG）
	cmd.Env = a.getCleanedEnvForVfox()

	out, err := cmd.CombinedOutput()

	// 去除输出中的 ANSI 转义字符（即颜色代码）
	cleanOut := ansiRegex.ReplaceAllString(string(out), "")

	if ctx.Err() == context.DeadlineExceeded {
		return cleanOut, fmt.Errorf("vfox %v timed out after 15s", args)
	}

	if err != nil {
		return cleanOut, fmt.Errorf("command failed: %w, output: %s", err, cleanOut)
	}

	return cleanOut, nil
}

// RunVfoxWithProgress 执行长耗时的 vfox 命令，将输出流实时发送到前端
func (a *App) RunVfoxWithProgress(args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, filepath.Join(a.getCoreDir(), getVfoxExeName()), args...)
	hideWindow(cmd)
	// 自动输入 "y" 以跳过任何预料之外的交互式确认 (最多输入5次y)
	cmd.Stdin = strings.NewReader("y\ny\ny\ny\ny\n")
	cmd.Env = a.getCleanedEnvForVfox()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		runtime.EventsEmit(a.ctx, "vfox-log", fmt.Sprintf("[EXIT ERROR] StdoutPipe failed: %v", err))
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		runtime.EventsEmit(a.ctx, "vfox-log", fmt.Sprintf("[EXIT ERROR] StderrPipe failed: %v", err))
		return err
	}

	if err := cmd.Start(); err != nil {
		runtime.EventsEmit(a.ctx, "vfox-log", fmt.Sprintf("[EXIT ERROR] cmd.Start failed: %v", err))
		return err
	}

	// 自定义拆分函数，遇到 \r 或 \n 都切分，防止进度条阻塞
	splitFunc := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	// 实时读取标准输出
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB 缓冲区
		scanner.Split(splitFunc)
		for scanner.Scan() {
			if a.ctx == nil {
				continue
			}
			line := scanner.Text()
			if line == "" {
				continue
			}
			cleanLine := ansiRegex.ReplaceAllString(line, "")
			runtime.EventsEmit(a.ctx, "vfox-log", cleanLine)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			runtime.EventsEmit(a.ctx, "vfox-log", "[STDOUT READ ERROR] "+err.Error())
		}
	}()

	// 实时读取标准错误
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB 缓冲区
		scanner.Split(splitFunc)
		for scanner.Scan() {
			if a.ctx == nil {
				continue
			}
			line := scanner.Text()
			if line == "" {
				continue
			}
			cleanLine := ansiRegex.ReplaceAllString(line, "")
			runtime.EventsEmit(a.ctx, "vfox-log", "[ERROR] "+cleanLine)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			runtime.EventsEmit(a.ctx, "vfox-log", "[STDERR READ ERROR] "+err.Error())
		}
	}()

	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		runtime.EventsEmit(a.ctx, "vfox-log", "[TIMEOUT] Command cancelled after 30min")
		return err
	}
	if err != nil {
		runtime.EventsEmit(a.ctx, "vfox-log", fmt.Sprintf("[EXIT ERROR] %v", err))
		return err
	}

	runtime.EventsEmit(a.ctx, "vfox-log", "[DONE]")
	return nil
}

// --- 数据结构 ---
type SdkVersion struct {
	Version string `json:"version"`
}

type SdkInfo struct {
	Name     string       `json:"name"`
	Source   string       `json:"source"` // "vfox" or "system"
	Path     string       `json:"path"`   // 绝对路径（主要针对系统 SDK，vfox SDK的路径动态获取）
	Versions []SdkVersion `json:"versions"`
}

type PluginInfo struct {
	Name       string `json:"name"`
	IsAdded    bool   `json:"isAdded"`
	IsOfficial bool   `json:"isOfficial"`
	URL        string `json:"url"`
}

// 版本控制相关数据结构
type SdkVersionDetail struct {
	Version   string `json:"version"`
	IsCurrent bool   `json:"isCurrent"`
}

type SdkDetail struct {
	Name     string             `json:"name"`
	Current  string             `json:"current"`
	Versions []SdkVersionDetail `json:"versions"`
}

// GetInstalledSdks 解析 vfox ls 的输出
func (a *App) GetInstalledSdks() ([]SdkInfo, error) {
	out, err := a.RunVfoxCommand("ls")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var sdks []SdkInfo
	var currentSdk *SdkInfo

	// 匹配插件名 (如 ├─┬golang 或 └─┬java，或者没有版本的 ├──php 或 └──php)
	pluginNameRegex := regexp.MustCompile(`^[├└]─[┬─](.+)`)
	// 匹配版本号 (如 │ └──1.26.3 或   ├──25.0.2+10)
	versionRegex := regexp.MustCompile(`^[│ ]\s*[├└]──(.*)`)

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if match := pluginNameRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				sdks = append(sdks, *currentSdk)
			}
			currentSdk = &SdkInfo{Name: match[1], Versions: []SdkVersion{}, Source: "vfox"}
		} else if match := versionRegex.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				v := match[1]
				if !strings.HasPrefix(v, "custom-sys-") {
					currentSdk.Versions = append(currentSdk.Versions, SdkVersion{Version: v})
				}
			}
		}
	}
	if currentSdk != nil {
		sdks = append(sdks, *currentSdk)
	}

	return sdks, nil
}

// getCacheFile 获取插件缓存文件的路径
func (a *App) getCacheFile() string {
	return filepath.Join(a.getVfoxHome(), "gui-plugins-cache.json")
}

func (a *App) getSystemSdkCacheFile() string {
	return filepath.Join(a.getVfoxHome(), "gui-system-sdks-cache.json")
}

func (a *App) getNonVfoxSdksFile() string {
	return filepath.Join(a.getVfoxHome(), "gui-non-vfox-sdks.json")
}

// GetAddedPlugins returns all plugins that have been added by parsing the plugin directory
func (a *App) GetAddedPlugins() ([]string, error) {
	vfoxHome := a.getVfoxHome()
	pluginDir := filepath.Join(vfoxHome, "plugin")

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			plugins = append(plugins, entry.Name())
		}
	}
	return plugins, nil
}

// GetAvailablePlugins 获取所有官方支持的插件
func (a *App) GetAvailablePlugins() ([]PluginInfo, error) {
	cacheFile := a.getCacheFile()
	var plugins []PluginInfo

	// 尝试从缓存读取
	data, err := os.ReadFile(cacheFile)
	if err == nil {
		if json.Unmarshal(data, &plugins) == nil && len(plugins) > 0 {
			// 更新 IsAdded 状态
			return a.applyIsAddedStatus(plugins), nil
		}
	}

	// 如果缓存不存在或为空，则强制刷新
	return a.RefreshAvailablePlugins()
}

// RefreshAvailablePlugins 强制执行 vfox available 并刷新缓存文件
func (a *App) RefreshAvailablePlugins() ([]PluginInfo, error) {
	out, err := a.RunVfoxCommand("available")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var plugins []PluginInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "AVAILABLE PLUGINS") || strings.HasPrefix(line, "Use 'vfox") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			name := parts[0]
			status := parts[1]
			url := parts[2]

			// ✓ (U+2713) or ✗ (U+2717)
			isOfficial := status == "✓"
			plugins = append(plugins, PluginInfo{
				Name:       name,
				IsOfficial: isOfficial,
				URL:        url,
			})
		}
	}

	// 写入缓存文件 (忽略写入错误，因为这只是缓存)
	if len(plugins) > 0 {
		if data, err := json.Marshal(plugins); err == nil {
			_ = os.WriteFile(a.getCacheFile(), data, 0644)
		}
	}

	return a.applyIsAddedStatus(plugins), nil
}

// applyIsAddedStatus 根据已安装 SDK 列表，刷新可用插件数组的 IsAdded 字段
func (a *App) applyIsAddedStatus(plugins []PluginInfo) []PluginInfo {
	installedSdks, _ := a.GetInstalledSdks()
	addedMap := make(map[string]bool)
	for _, sdk := range installedSdks {
		if sdk.Source == "vfox" {
			addedMap[sdk.Name] = true
		}
	}
	for i := range plugins {
		plugins[i].IsAdded = addedMap[plugins[i].Name]
	}
	return plugins
}

// --- 版本控制方法 ---

// GetSdkDetail 获取单个 SDK 的详情，包含版本列表和当前版本标记
func (a *App) GetSdkDetail(name string) (SdkDetail, error) {
	currentOut, _ := a.RunVfoxCommand("current", name)
	currentVer := strings.TrimPrefix(strings.TrimSpace(currentOut), "-> ")
	// vfox current 输出带 v 前缀 (如 v1.26.3)，去掉以匹配 ls 输出
	currentVer = strings.TrimPrefix(strings.TrimSpace(currentVer), "v")

	// Fallback: if vfox current fails (e.g. for custom-sys- versions), read from .vfox.toml directly
	if currentVer == "" || strings.Contains(currentVer, "no current") {
		tomlPath := filepath.Join(a.getVfoxHome(), ".vfox.toml")
		if data, err := os.ReadFile(tomlPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, name+" ") || strings.HasPrefix(trimmed, name+"=") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						v := strings.TrimSpace(parts[1])
						v = strings.Trim(v, "\"")
						currentVer = v
					}
					break
				}
			}
		}
	}

	if strings.Contains(currentVer, "no current") {
		currentVer = ""
	}

	out, err := a.RunVfoxCommand("ls", name)
	if err != nil {
		// 如果 ls 失败，直接按 0 个版本处理，但需要带上 currentVer
		return SdkDetail{Name: name, Versions: make([]SdkVersionDetail, 0), Current: currentVer}, nil
	}

	lines := strings.Split(out, "\n")
	detail := SdkDetail{Name: name, Current: currentVer}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		ver := strings.TrimPrefix(line, "-> ")
		isCurrent := strings.HasSuffix(ver, " <— current")
		ver = strings.TrimSuffix(ver, " <— current")
		ver = strings.TrimSpace(ver)

		if ver == "" || strings.Contains(ver, "installed sdk") || strings.HasPrefix(ver, "custom-sys-") {
			continue
		}

		detail.Versions = append(detail.Versions, SdkVersionDetail{
			Version:   ver,
			IsCurrent: isCurrent,
		})
	}

	return detail, nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// UseVersion 切换到指定版本
func (a *App) UseVersion(name, version string) (string, error) {
	// 异步执行避免 RPC 卡住，前端通过 sdk-list-changed 事件感知完成
	go func() {
		// 使用 RunVfoxCommand 正常等待 vfox use 完成，因为我们已经在环境变量里伪装成了 cmd
		// 这样它就不会死锁了！
		// 注意：不要在 vfox use 之后手动覆盖 junction！
		// vfox 会自动创建正确的 junction（指向如 .../v-3.12.7/python-3.12.7），
		// 而 GetVersionPath 返回的是上一级目录（.../v-3.12.7），如果我们覆盖
		// 会导致 PATH 多出一层目录，SDK 实际上无法被找到。
		_, err := a.RunVfoxCommand("use", "--global", name+"@"+version)

		if a.ctx != nil {
			if err != nil {
				runtime.EventsEmit(a.ctx, "vfox-log", "[ERROR] "+err.Error())
			}
			runtime.EventsEmit(a.ctx, "vfox-log", "[DONE]")
			runtime.EventsEmit(a.ctx, "sdk-list-changed")
		}
	}()
	return "ok", nil
}

// getSdkRoot resolves the SDK root directory from an executable path.
// If the exe is inside bin/sbin/scripts, it goes up one level.
func (a *App) getSdkRoot(exePath string) string {
	dir := filepath.Dir(exePath)
	base := strings.ToLower(filepath.Base(dir))
	if base == "bin" || base == "sbin" || base == "scripts" {
		return filepath.Dir(dir)
	}
	return dir
}

// removeJunctionIfExists removes a junction/directory if it exists.
func (a *App) removeJunctionIfExists(linkPath string) {
	_ = os.RemoveAll(linkPath)
}

// UseCustomSdk bypasses `vfox use` entirely because Go's os.ReadDir reports
// Windows junctions as ModeIrregular (not directories), which causes vfox's
// GetRuntimePackage to fail with "runtime not found".
//
// Instead, we directly:
// 1. Clear vfox's current selection via `vfox unuse`
// 2. Create a junction/symlink at ~/.vfox/sdks/{name} -> system SDK root
func (a *App) UseCustomSdk(name string, exePath string) (string, error) {
	root := a.getSdkRoot(exePath)

	// 1. Clear the vfox selection so that both don't show "当前" at the same time
	// We MUST do this synchronously BEFORE overwriting the symlink, otherwise vfox might delete our custom symlink
	if _, err := a.RunVfoxCommand("unuse", "--global", name); err != nil {
		// Non-fatal: log and continue — the junction creation is more important
		fmt.Printf("[warn] vfox unuse %s failed: %v\n", name, err)
	}

	// 2. Create junction: ~/.vfox/sdks/{name} -> system SDK root
	sdkLinkPath := filepath.Join(a.getVfoxHome(), "sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, root); err != nil {
		return "", fmt.Errorf("failed to create SDK junction: %v", err)
	}

	go func() {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "vfox-log", fmt.Sprintf("Activating %s (system)...", name))
			runtime.EventsEmit(a.ctx, "vfox-log", "[DONE]")
			runtime.EventsEmit(a.ctx, "sdk-list-changed")
		}
	}()

	return "ok", nil
}

// UnuseVersion 取消当前 SDK 的版本设置（异步，避免 RPC 阻塞）
func (a *App) UnuseVersion(name string) (string, error) {
	go func() {
		// Clean up junction before unuse
		sdkLinkPath := filepath.Join(a.getVfoxHome(), "sdks", name)
		a.removeJunctionIfExists(sdkLinkPath)

		_, err := a.RunVfoxCommand("unuse", "--global", name)
		if a.ctx != nil {
			if err != nil {
				runtime.EventsEmit(a.ctx, "vfox-log", "[ERROR] "+err.Error())
			}
			runtime.EventsEmit(a.ctx, "vfox-log", "[DONE]")
			runtime.EventsEmit(a.ctx, "sdk-list-changed")
		}
	}()
	return "ok", nil
}

// InstallVersion 安装指定版本 (会耗时，并且会产生 vfox-log 进度事件)
func (a *App) InstallVersion(name, version string) error {
	return a.RunVfoxWithProgress([]string{"install", "-y", name + "@" + version})
}

// RemovePlugin 移除指定插件及其相关的 SDK 和环境变量
func (a *App) RemovePlugin(name string) error {
	// 1. 先获取该插件所有已安装的版本
	sdks, err := a.GetInstalledSdks()
	if err == nil {
		for _, sdk := range sdks {
			if sdk.Name == name && sdk.Source == "vfox" {
				// 2. 逐个卸载版本 (vfox uninstall 会顺带清理它建立的链接和环境配置)
				for _, v := range sdk.Versions {
					_ = a.RunVfoxWithProgress([]string{"uninstall", name + "@" + v.Version})
				}
				break
			}
		}
	}

	// 3. 执行 vfox unuse 以确保没有任何全局/局部环境残留 (防范于未然，忽略错误)
	_, _ = a.RunVfoxCommand("unuse", "-g", name)
	_, _ = a.RunVfoxCommand("unuse", "-p", name)
	_, _ = a.RunVfoxCommand("unuse", "-s", name)

	// 4. 彻底删除插件
	err = a.RunVfoxWithProgress([]string{"remove", "-y", name})

	// 5. Delete all associated custom SDKs (None Vfox SDKs)
	m := a.GetNonVfoxSdksMap()
	if _, ok := m[name]; ok {
		delete(m, name)
		a.saveNonVfoxSdksMap(m)
	}

	if err == nil {
		// 6. 保底清理：防止因任何异常导致残留的 SDK 空目录
		vfoxHome := a.getVfoxHome()
		if vfoxHome != "" {
			vfoxLinkPath := filepath.Join(vfoxHome, "sdks", name)
			_ = os.RemoveAll(vfoxLinkPath)
		}
	}
	return err
}

// UninstallVersion 卸载指定版本
func (a *App) UninstallVersion(name, version string) error {
	return a.RunVfoxWithProgress([]string{"uninstall", name + "@" + version})
}

// GetVersionPath 获取指定版本的 SDK 绝对安装路径
func (a *App) GetVersionPath(name, version string) (string, error) {
	out, err := a.RunVfoxCommand("info", name+"@"+version)
	if err != nil {
		return "", err
	}

	// vfox info 的最后一行或者非空行即为路径。通常情况下输出的就是绝对路径。
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[len(lines)-1]), nil
	}
	return "", nil
}

// SearchVersions 搜索 SDK 的可用版本，网络错误时返回空列表
func (a *App) SearchVersions(name string) ([]string, error) {
	out, err := a.RunVfoxCommand("search", name)
	if err != nil {
		// search 可能因网络超时失败，返回空列表而非报错
		return []string{}, nil
	}

	lines := strings.Split(out, "\n")
	var versions []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		// 过滤掉表头或说明行
		if strings.HasPrefix(line, "Available") ||
			strings.HasPrefix(line, "Search") ||
			strings.HasPrefix(line, "Please") ||
			strings.HasPrefix(line, "Use") ||
			strings.HasPrefix(line, "Name") ||
			strings.HasPrefix(line, "---") {
			continue
		}

		// 找到第一个包含字母或数字的字段作为版本号
		parts := strings.Fields(line)
		var ver string
		for _, p := range parts {
			hasAlphaNum := false
			for _, r := range p {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					hasAlphaNum = true
					break
				}
			}
			if hasAlphaNum {
				ver = p
				break
			}
		}

		if ver != "" && ver != "Error:" && ver != "Available" {
			// 避免提取到一些表头词汇，如果在前置过滤里没过滤掉的话
			versions = append(versions, ver)
		}
	}

	return versions, nil
}

// --- 系统 SDK 扫描（并行 + 缓存） ---

type systemSDKDef struct {
	Name    string
	Exe     string
	VerArgs []string
}

var systemSDKDefs = []systemSDKDef{
	{"python", "python", []string{"--version"}},
	{"nodejs", "node", []string{"--version"}},
	{"java", "java", []string{"-version"}},
	{"golang", "go", []string{"version"}},
	{"rust", "rustc", []string{"--version"}},
	{"dotnet", "dotnet", []string{"--version"}},
	{"ruby", "ruby", []string{"--version"}},
	{"php", "php", []string{"--version"}},
	{"perl", "perl", []string{"--version"}},
	{"git", "git", []string{"--version"}},
	{"docker", "docker", []string{"--version"}},
	{"zig", "zig", []string{"version"}},
	{"lua", "lua", []string{"-v"}},
	{"gcc", "gcc", []string{"--version"}},
	{"cmake", "cmake", []string{"--version"}},
}

var (
	systemSdkCache   []SdkInfo
	systemSdkCacheMu sync.RWMutex
)

// GetCachedSystemSdks returns a copy of the cached system SDK list, never blocking
func (a *App) GetCachedSystemSdks() []SdkInfo {
	systemSdkCacheMu.RLock()
	defer systemSdkCacheMu.RUnlock()
	result := make([]SdkInfo, len(systemSdkCache))
	copy(result, systemSdkCache)
	return result
}

// ScanSystemSdks 并行扫描并更新缓存
func (a *App) ScanSystemSdks() {
	// First, try loading from cache file to populate UI instantly
	cacheFile := a.getSystemSdkCacheFile()
	if data, err := os.ReadFile(cacheFile); err == nil {
		var cachedResult []SdkInfo
		if err := json.Unmarshal(data, &cachedResult); err == nil && len(cachedResult) > 0 {
			systemSdkCacheMu.Lock()
			systemSdkCache = cachedResult
			systemSdkCacheMu.Unlock()
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "system-sdks-ready")
			}
		}
	}

	// Build a clean PATH that excludes .vfox entries, for child processes only.
	// IMPORTANT: Do NOT use os.Setenv here — it modifies global process state
	// and races with concurrent goroutines (RunVfoxCommand, RunVfoxWithProgress, etc.).
	originalPath := os.Getenv("PATH")
	paths := filepath.SplitList(originalPath)
	var cleanPaths []string
	for _, p := range paths {
		if !strings.Contains(strings.ToLower(p), ".vfox") {
			cleanPaths = append(cleanPaths, p)
		}
	}
	cleanPathStr := strings.Join(cleanPaths, string(filepath.ListSeparator))

	// Build clean env slice for child processes
	baseEnv := os.Environ()
	var cleanEnv []string
	for _, e := range baseEnv {
		if strings.HasPrefix(strings.ToLower(e), "path=") {
			cleanEnv = append(cleanEnv, "PATH="+cleanPathStr)
		} else {
			cleanEnv = append(cleanEnv, e)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var result []SdkInfo

	for _, def := range systemSDKDefs {
		wg.Add(1)
		go func(d systemSDKDef) {
			defer wg.Done()
			exePath := findExecutable(d.Exe, cleanEnv)
			if exePath == "" {
				return
			}

			ver := a.tryGetVersionWithEnv(d.Exe, d.VerArgs, cleanEnv)
			if ver == "" {
				ver = "unknown"
			}
			mu.Lock()
			result = append(result, SdkInfo{
				Name:     d.Name,
				Source:   "system",
				Path:     exePath,
				Versions: []SdkVersion{{Version: ver}},
			})
			mu.Unlock()
		}(def)
	}
	wg.Wait()

	systemSdkCacheMu.Lock()
	systemSdkCache = result
	systemSdkCacheMu.Unlock()

	// Write updated results to cache file
	if data, err := json.Marshal(result); err == nil {
		_ = os.WriteFile(cacheFile, data, 0644)
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "system-sdks-ready")
	}
}

// GetNonVfoxSdksMap returns the persisted map of custom SDKs
func (a *App) GetNonVfoxSdksMap() map[string][]SdkInfo {
	f := a.getNonVfoxSdksFile()
	data, err := os.ReadFile(f)
	res := make(map[string][]SdkInfo)
	if err == nil {
		_ = json.Unmarshal(data, &res)
	}
	return res
}

func (a *App) saveNonVfoxSdksMap(m map[string][]SdkInfo) {
	if data, err := json.Marshal(m); err == nil {
		_ = os.WriteFile(a.getNonVfoxSdksFile(), data, 0644)
	}
}

// GetNonVfoxSdks exposes the full non-vfox list to the frontend
func (a *App) GetNonVfoxSdks() map[string][]SdkInfo {
	return a.GetNonVfoxSdksMap()
}

// DetectSdkPathVersion tries to extract version from a given custom sdk path
func (a *App) DetectSdkPathVersion(name string, exePath string) string {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return "unknown"
	}
	for _, def := range systemSDKDefs {
		if def.Name == name {
			v := a.tryGetVersion(exePath, def.VerArgs)
			if v != "" {
				return v
			}
			break
		}
	}
	return "unknown"
}

// AddNonVfoxSdk allows manual registration of a non-vfox SDK path
func (a *App) AddNonVfoxSdk(name string, exePath string, version string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if version == "" {
		version = "unknown"
	}

	m := a.GetNonVfoxSdksMap()
	list := m[name]
	for _, existing := range list {
		if existing.Path == exePath {
			return fmt.Errorf("path already exists")
		}
	}
	m[name] = append(m[name], SdkInfo{
		Name:     name,
		Source:   "system",
		Path:     exePath,
		Versions: []SdkVersion{{Version: version}},
	})
	a.saveNonVfoxSdksMap(m)
	return nil
}

// RemoveNonVfoxSdk removes a custom path from the non-vfox list
func (a *App) RemoveNonVfoxSdk(name string, exePath string) error {
	m := a.GetNonVfoxSdksMap()
	list := m[name]
	var newList []SdkInfo
	for _, existing := range list {
		if existing.Path != exePath {
			newList = append(newList, existing)
		} else {
			// Only remove the sdk symlink if it currently points to the SDK being removed
			activePath, _ := a.GetActiveCustomSdk(name)
			if strings.EqualFold(filepath.Clean(activePath), filepath.Clean(exePath)) {
				sdkLinkPath := filepath.Join(a.getVfoxHome(), "sdks", name)
				a.removeJunctionIfExists(sdkLinkPath)
			}
		}
	}
	if len(newList) == len(list) {
		return fmt.Errorf("path not found")
	}
	if len(newList) == 0 {
		delete(m, name)
	} else {
		m[name] = newList
	}
	a.saveNonVfoxSdksMap(m)
	return nil
}

func (a *App) tryGetVersion(exe string, args []string) string {
	return a.tryGetVersionWithEnv(exe, args, nil)
}

// tryGetVersionWithEnv runs the version command with an optional custom environment.
// If env is nil, inherits the current process environment.
func (a *App) tryGetVersionWithEnv(exe string, args []string, env []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, args...)
	hideWindow(cmd)
	if env != nil {
		cmd.Env = env
	}
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		return ""
	}
	return extractVersion(string(out), exe)
}

func extractVersion(raw string, exe string) string {
	clean := ansiRegex.ReplaceAllString(raw, "")
	clean, _, _ = strings.Cut(clean, "\n")
	clean = strings.TrimRight(clean, "\r")
	clean = strings.TrimSpace(clean)

	switch exe {
	case "python":
		return strings.TrimPrefix(clean, "Python ")
	case "node":
		return strings.TrimPrefix(clean, "v")
	case "java":
		// java -version 输出到 stderr，格式: openjdk version "21.0.2"
		if idx := strings.Index(clean, `"`); idx >= 0 {
			rest := clean[idx+1:]
			if idx2 := strings.Index(rest, `"`); idx2 >= 0 {
				return rest[:idx2]
			}
		}
		return clean
	case "go":
		parts := strings.Fields(clean)
		if len(parts) >= 3 {
			return strings.TrimPrefix(parts[2], "go")
		}
		return clean
	case "rustc", "ruby", "php", "lua":
		parts := strings.Fields(clean)
		if len(parts) >= 2 {
			return parts[1]
		}
		return clean
	case "dotnet", "zig":
		return clean
	case "perl":
		parts := strings.Fields(clean)
		for i, p := range parts {
			if p == "version" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		return clean
	case "git":
		return strings.TrimPrefix(clean, "git version ")
	case "docker":
		parts := strings.Fields(clean)
		if len(parts) >= 3 {
			return strings.TrimSuffix(parts[2], ",")
		}
		return clean
	case "gcc":
		parts := strings.Fields(clean)
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return clean
	case "cmake":
		return strings.TrimPrefix(clean, "cmake version ")
	default:
		return clean
	}
}

// GetAllSdks 合并 vfox SDK（实时）和系统自动检测 SDK（缓存），去重返回
func (a *App) GetAllSdks() ([]SdkInfo, error) {
	vfoxSdks, err := a.GetInstalledSdks()
	if err != nil {
		vfoxSdks = []SdkInfo{}
	}

	cached := a.GetCachedSystemSdks()

	seen := make(map[string]bool)
	var result []SdkInfo

	for _, s := range vfoxSdks {
		seen[s.Name] = true
		result = append(result, s)
	}
	for _, s := range cached {
		if !seen[s.Name] {
			result = append(result, s)
		}
	}

	return result, nil
}

// getCoreDir returns the absolute path to the "core" directory containing vfox[.exe].
// Search order:
//  1. {exe_dir}/core/           — Windows NSIS install & dev mode
//  2. {exe_dir}/../Resources/core/ — macOS .app bundle (Contents/MacOS/../Resources/core)
//  3. /usr/lib/vfoxg/core/      — Linux DEB/RPM system install
//  4. {cwd}/core/               — dev fallback
func (a *App) getCoreDir() string {
	// Map Go runtime constants to actual core directory names
	osName := stdruntime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	archName := stdruntime.GOARCH
	switch archName {
	case "amd64":
		archName = "x86_64"
	case "386":
		archName = "x86"
	// arm64 stays as-is
	}
	suffix := filepath.Join(osName, archName)

	// Build candidate list
	var candidates []string

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		// 1. {exe_dir}/core/ — standard for Windows installer
		candidates = append(candidates, filepath.Join(exeDir, "core"))
		// 2. {exe_dir}/../Resources/core/ — macOS .app bundle
		candidates = append(candidates, filepath.Join(exeDir, "..", "Resources", "core"))
	}

	// 3. /usr/lib/vfoxg/core/ — Linux system package
	if stdruntime.GOOS == "linux" {
		candidates = append(candidates, "/usr/lib/vfoxg/core")
	}

	// 4. {cwd}/core/ — dev fallback
	if abs, err := filepath.Abs("core"); err == nil {
		candidates = append(candidates, abs)
	}

	for _, c := range candidates {
		full := filepath.Join(c, suffix)
		if _, err := os.Stat(full); err == nil {
			return full
		}
	}

	// Ultimate fallback (may not exist, but avoids empty string)
	baseDir, _ := filepath.Abs("core")
	return filepath.Join(baseDir, suffix)
}

// GetActiveCustomSdk reads the junction target. If it points to an external path (not containing .vfox/cache),
// it's a Custom SDK and returns the path.
func (a *App) GetActiveCustomSdk(name string) (string, error) {
	sdkLinkPath := filepath.Join(a.getVfoxHome(), "sdks", name)

	fi, err := os.Lstat(sdkLinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0 {
		target, err := os.Readlink(sdkLinkPath)
		if err == nil {
			// If target contains .vfox/cache, it's a vfox managed SDK
			if strings.Contains(filepath.ToSlash(target), "/.vfox/cache/") {
				return "", nil
			}
			// Try to find matching SDK path
			m := a.GetNonVfoxSdksMap()
			cleanTarget := filepath.Clean(target)
			for _, sdk := range m[name] {
				if strings.EqualFold(filepath.Clean(a.getSdkRoot(sdk.Path)), cleanTarget) {
					return sdk.Path, nil
				}
			}
			return target, nil
		}
	}

	return "", nil
}

func (a *App) getVfoxHome() string {
	if v := os.Getenv("VFOX_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".vfox")
	}
	return ""
}
