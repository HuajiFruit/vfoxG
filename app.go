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
	ctx      context.Context
	homeMu   sync.RWMutex
	vfoxHome string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.loadVfoxHomeSetting(); err != nil {
		a.emitEvent("vfox-log", "[ERROR] "+err.Error())
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		a.emitEvent("vfox-log", "[ERROR] "+err.Error())
	}
	go a.ScanSystemSdks()
	go a.RefreshAvailablePlugins() // 后台预热插件市场缓存
}

// getCleanedEnvForVfox returns a copy of the current environment but sanitizes the PATH
// to remove any previously injected vfox sdks, ensuring `vfox` reads from .vfox.toml.
func (a *App) getCleanedEnvForVfox() []string {
	env := os.Environ()
	vfoxHome := strings.TrimSpace(a.getVfoxHome())
	vfoxSdksDir := a.getVfoxHomePath("sdks")
	sep := string(filepath.ListSeparator) // ";" on Windows, ":" on Unix

	if vfoxSdksDir != "" {
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
					if isPathWithin(pTrim, vfoxSdksDir) {
						continue
					}
					clean = append(clean, pTrim)
				}
				env[i] = "PATH=" + strings.Join(clean, sep)
				break
			}
		}
	}
	// 添加伪装变量，使用 cmd/bash 可以避免 vfox 弹出子 shell 而导致死锁
	shellName := "bash"
	if stdruntime.GOOS == "windows" {
		shellName = "cmd"
	}
	env = upsertEnv(env, "VFOX_HOME", vfoxHome)
	env = upsertEnv(env, "__VFOX_SHELL", shellName)
	return env
}

// isPathWithin reports whether path is root itself or a child of root.
func isPathWithin(path string, root string) bool {
	path = strings.TrimSpace(path)
	root = strings.TrimSpace(root)
	if path == "" || root == "" {
		return false
	}

	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if stdruntime.GOOS == "windows" {
		path = strings.ToLower(path)
		root = strings.ToLower(root)
	}
	if path == root {
		return true
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// RunVfoxCommand 执行短生命周期的 vfox 命令，带 15s 超时防止前端卡死
func (a *App) RunVfoxCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	vfoxExe, err := a.getVfoxExecutable()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, vfoxExe, args...)

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

	vfoxExe, err := a.getVfoxExecutable()
	if err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return err
	}

	cmd := exec.CommandContext(ctx, vfoxExe, args...)
	hideWindow(cmd)
	// 自动输入 "y" 以跳过任何预料之外的交互式确认 (最多输入5次y)
	cmd.Stdin = strings.NewReader("y\ny\ny\ny\ny\n")
	cmd.Env = a.getCleanedEnvForVfox()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] StdoutPipe failed: %v", err))
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] StderrPipe failed: %v", err))
		return err
	}

	if err := cmd.Start(); err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] cmd.Start failed: %v", err))
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
			a.emitEvent("vfox-log", cleanLine)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			a.emitEvent("vfox-log", "[STDOUT READ ERROR] "+err.Error())
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
			a.emitEvent("vfox-log", "[ERROR] "+cleanLine)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			a.emitEvent("vfox-log", "[STDERR READ ERROR] "+err.Error())
		}
	}()

	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		a.emitEvent("vfox-log", "[TIMEOUT] Command cancelled after 30min")
		if err != nil {
			return fmt.Errorf("vfox %v timed out after 30min: %w", args, err)
		}
		return fmt.Errorf("vfox %v timed out after 30min", args)
	}
	if err != nil {
		a.emitEvent("vfox-log", fmt.Sprintf("[EXIT ERROR] %v", err))
		return err
	}

	a.emitEvent("vfox-log", "[DONE]")
	return nil
}

func (a *App) emitEvent(name string, data ...interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data...)
}

func upsertEnv(env []string, key string, value string) []string {
	prefix := key + "="
	for i, e := range env {
		name, _, ok := strings.Cut(e, "=")
		if ok && envKeyEqual(name, key) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func envKeyEqual(a string, b string) bool {
	if stdruntime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func (a *App) appConfigFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("unable to resolve user config directory")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "vfoxG", "config.json"), nil
}

func (a *App) readAppConfig() (AppConfig, error) {
	configFile, err := a.appConfigFile()
	if err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return AppConfig{}, nil
		}
		return AppConfig{}, err
	}
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return AppConfig{}, err
	}
	return config, nil
}

func (a *App) saveAppConfig(config AppConfig) error {
	configFile, err := a.appConfigFile()
	if err != nil {
		return err
	}
	return a.writeJSONFile(configFile, config)
}

func (a *App) loadVfoxHomeSetting() error {
	config, err := a.readAppConfig()
	if err != nil {
		return err
	}
	path := strings.TrimSpace(config.VfoxHome)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("VFOX_HOME"))
	}
	if path == "" {
		path = a.defaultVfoxHome()
	}
	normalized, err := normalizeDownloadPath(path)
	if err != nil {
		return err
	}
	a.setVfoxHome(normalized)
	return nil
}

func (a *App) setVfoxHome(path string) {
	a.homeMu.Lock()
	a.vfoxHome = path
	a.homeMu.Unlock()
}

func (a *App) appInstallDir() string {
	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		if stdruntime.GOOS == "darwin" && filepath.Base(exeDir) == "MacOS" {
			contentsDir := filepath.Dir(exeDir)
			if filepath.Base(contentsDir) == "Contents" {
				return filepath.Dir(filepath.Dir(contentsDir))
			}
		}
		return exeDir
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (a *App) defaultVfoxHome() string {
	if path, err := defaultUserVfoxHome(); err == nil {
		return path
	}
	return filepath.Join(os.TempDir(), "vfoxG", "vfox-home")
}

func defaultUserVfoxHome() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || strings.TrimSpace(home) == "" {
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("unable to resolve user home directory")
		}
		base = filepath.Join(home, ".config")
	}
	return normalizeDownloadPath(filepath.Join(base, "vfoxG", "vfox-home"))
}

func normalizeDownloadPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("download path cannot be empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", fmt.Errorf("cannot expand home directory")
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimLeft(path[1:], `/\`))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func (a *App) GetDownloadPathInfo() (DownloadPathInfo, error) {
	path := a.getVfoxHome()
	defaultPath := a.defaultVfoxHome()
	return DownloadPathInfo{
		Path:        path,
		DefaultPath: defaultPath,
		IsDefault:   samePath(path, defaultPath),
	}, nil
}

func (a *App) SetDownloadPath(path string) (DownloadPathInfo, error) {
	normalized, err := normalizeDownloadPath(path)
	if err != nil {
		return DownloadPathInfo{}, err
	}
	if err := os.MkdirAll(normalized, 0755); err != nil {
		return DownloadPathInfo{}, err
	}
	if err := a.saveAppConfig(AppConfig{VfoxHome: normalized}); err != nil {
		return DownloadPathInfo{}, err
	}
	a.setVfoxHome(normalized)
	a.emitEvent("vfox-log", "[INFO] VFOX_HOME="+normalized)
	a.emitEvent("vfox-home-changed")
	a.emitEvent("sdk-list-changed")
	go a.RefreshAvailablePlugins()
	go a.ScanSystemSdks()
	return a.GetDownloadPathInfo()
}

func (a *App) ResetDownloadPath() (DownloadPathInfo, error) {
	defaultPath := a.defaultVfoxHome()
	if err := os.MkdirAll(defaultPath, 0755); err != nil {
		return DownloadPathInfo{}, err
	}
	if err := a.saveAppConfig(AppConfig{}); err != nil {
		return DownloadPathInfo{}, err
	}
	a.setVfoxHome(defaultPath)
	a.emitEvent("vfox-log", "[INFO] VFOX_HOME="+defaultPath)
	a.emitEvent("vfox-home-changed")
	a.emitEvent("sdk-list-changed")
	go a.RefreshAvailablePlugins()
	go a.ScanSystemSdks()
	return a.GetDownloadPathInfo()
}

func (a *App) SelectDownloadPath() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}
	current := a.getVfoxHome()
	defaultDir := current
	if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
		defaultDir = filepath.Dir(defaultDir)
		if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
			defaultDir = a.appInstallDir()
		}
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select SDK and plugin download directory",
		DefaultDirectory:     defaultDir,
		CanCreateDirectories: true,
	})
}

func samePath(a string, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if stdruntime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
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

type AppConfig struct {
	VfoxHome string `json:"vfoxHome"`
}

func customSdksToKeep(list []SdkInfo, keepPath string) ([]SdkInfo, error) {
	keepPath = strings.TrimSpace(keepPath)
	if keepPath == "" {
		return nil, nil
	}
	for _, sdk := range list {
		if samePath(sdk.Path, keepPath) {
			return []SdkInfo{sdk}, nil
		}
	}
	return nil, fmt.Errorf("custom SDK path is not registered: %s", keepPath)
}

type DownloadPathInfo struct {
	Path        string `json:"path"`
	DefaultPath string `json:"defaultPath"`
	IsDefault   bool   `json:"isDefault"`
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

type PlatformInfo struct {
	OS                  string `json:"os"`
	Name                string `json:"name"`
	CoreOS              string `json:"coreOS"`
	CoreArch            string `json:"coreArch"`
	DownloadPath        string `json:"downloadPath"`
	DefaultDownloadPath string `json:"defaultDownloadPath"`
	VfoxPathTarget      string `json:"vfoxPathTarget"`
	SDKPathTarget       string `json:"sdkPathTarget"`
	ShellProfile        string `json:"shellProfile"`
	RequiresElevation   bool   `json:"requiresElevation"`
	RestartHint         string `json:"restartHint"`
}

func (a *App) GetPlatformInfo() PlatformInfo {
	info := PlatformInfo{
		OS:                  stdruntime.GOOS,
		Name:                stdruntime.GOOS,
		CoreOS:              coreOSName(),
		CoreArch:            coreArchName(),
		DownloadPath:        a.getVfoxHome(),
		DefaultDownloadPath: a.defaultVfoxHome(),
	}

	switch stdruntime.GOOS {
	case "windows":
		info.Name = "Windows"
		info.VfoxPathTarget = "User PATH"
		info.SDKPathTarget = "Machine PATH"
		info.ShellProfile = "Windows environment variables"
		info.RequiresElevation = true
		info.RestartHint = "Open a new terminal after changing PATH."
	case "darwin":
		info.Name = "macOS"
		info.VfoxPathTarget = "~/.zprofile"
		info.SDKPathTarget = "~/.zprofile"
		info.ShellProfile = displayHomePath(".zprofile")
		info.RestartHint = "Open a new terminal or run source ~/.zprofile."
	case "linux":
		info.Name = "Linux"
		info.VfoxPathTarget = "~/.profile"
		info.SDKPathTarget = "~/.profile"
		info.ShellProfile = displayHomePath(".profile")
		info.RestartHint = "Open a new terminal or run source ~/.profile."
	default:
		info.VfoxPathTarget = "user shell profile"
		info.SDKPathTarget = "user shell profile"
		info.ShellProfile = displayHomePath(".profile")
		info.RestartHint = "Open a new terminal after changing PATH."
	}

	return info
}

func displayHomePath(elem string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, elem)
	}
	return filepath.Join("~", elem)
}

func (a *App) CheckPluginPathOverride(pluginName string) bool {
	return a.CheckPluginWin11CompatMode(pluginName)
}

func (a *App) CheckAnyPathOverride() bool {
	return a.CheckWin11CompatMode()
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
	return a.getVfoxHomePath("gui-plugins-cache.json")
}

func (a *App) getSystemSdkCacheFile() string {
	return a.getVfoxHomePath("gui-system-sdks-cache.json")
}

func (a *App) getNonVfoxSdksFile() string {
	return a.getVfoxHomePath("gui-non-vfox-sdks.json")
}

func (a *App) getVfoxHomePath(elem ...string) string {
	vfoxHome := strings.TrimSpace(a.getVfoxHome())
	if vfoxHome == "" {
		return ""
	}
	parts := append([]string{vfoxHome}, elem...)
	return filepath.Join(parts...)
}

func (a *App) ensureVfoxHomeDir() error {
	vfoxHome := a.getVfoxHome()
	if strings.TrimSpace(vfoxHome) == "" {
		return fmt.Errorf("unable to resolve vfox home directory")
	}
	return os.MkdirAll(vfoxHome, 0755)
}

func (a *App) writeJSONFile(path string, v interface{}) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func validateSDKExecutablePath(exePath string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("path cannot be empty")
	}
	info, err := os.Stat(exePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", exePath)
		}
		return fmt.Errorf("cannot access path %s: %w", exePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path must point to an executable file, got directory: %s", exePath)
	}
	return nil
}

// GetAddedPlugins returns all plugins that have been added by parsing the plugin directory
func (a *App) GetAddedPlugins() ([]string, error) {
	vfoxHome := a.getVfoxHome()
	if strings.TrimSpace(vfoxHome) == "" {
		return []string{}, fmt.Errorf("unable to resolve vfox home directory")
	}
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
		_ = a.writeJSONFile(a.getCacheFile(), plugins)
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
		if data, err := os.ReadFile(a.getVfoxHomePath(".vfox.toml")); err == nil {
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

		if err != nil {
			a.emitEvent("vfox-log", "[ERROR] "+err.Error())
		}
		a.emitEvent("vfox-log", "[DONE]")
		a.emitEvent("sdk-list-changed")
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
	if err := validateSDKExecutablePath(exePath); err != nil {
		return "", err
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return "", err
	}
	root := a.getSdkRoot(exePath)

	// 1. Clear the vfox selection so that both don't show "当前" at the same time
	// We MUST do this synchronously BEFORE overwriting the symlink, otherwise vfox might delete our custom symlink
	if _, err := a.RunVfoxCommand("unuse", "--global", name); err != nil {
		// Non-fatal: log and continue — the junction creation is more important
		fmt.Printf("[warn] vfox unuse %s failed: %v\n", name, err)
	}

	// 2. Create junction: ~/.vfox/sdks/{name} -> system SDK root
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, root); err != nil {
		return "", fmt.Errorf("failed to create SDK junction: %v", err)
	}

	go func() {
		a.emitEvent("vfox-log", fmt.Sprintf("Activating %s (system)...", name))
		a.emitEvent("vfox-log", "[DONE]")
		a.emitEvent("sdk-list-changed")
	}()

	return "ok", nil
}

// UnuseVersion 取消当前 SDK 的版本设置（异步，避免 RPC 阻塞）
func (a *App) UnuseVersion(name string) (string, error) {
	go func() {
		// Clean up junction before unuse
		sdkLinkPath := a.getVfoxHomePath("sdks", name)
		a.removeJunctionIfExists(sdkLinkPath)

		_, err := a.RunVfoxCommand("unuse", "--global", name)
		if err != nil {
			a.emitEvent("vfox-log", "[ERROR] "+err.Error())
		}
		a.emitEvent("vfox-log", "[DONE]")
		a.emitEvent("sdk-list-changed")
	}()
	return "ok", nil
}

// InstallVersion 安装指定版本 (会耗时，并且会产生 vfox-log 进度事件)
func (a *App) InstallVersion(name, version string) error {
	return a.RunVfoxWithProgress([]string{"install", "-y", name + "@" + version})
}

// RemovePlugin 移除指定插件及其相关的 SDK 和环境变量
func (a *App) RemovePlugin(name string) error {
	return a.RemovePluginWithOptions(name, "")
}

// RemovePluginWithOptions removes a plugin and optionally keeps one custom SDK
// path active as the system SDK path override.
func (a *App) RemovePluginWithOptions(name string, keepCustomSdkPath string) (err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	keepCustomSdkPath = strings.TrimSpace(keepCustomSdkPath)

	m := a.GetNonVfoxSdksMap()
	keptCustomSdks, err := customSdksToKeep(m[name], keepCustomSdkPath)
	if err != nil {
		return err
	}
	if len(keptCustomSdks) > 0 {
		if err := validateSDKExecutablePath(keptCustomSdks[0].Path); err != nil {
			return err
		}
	}
	if len(keptCustomSdks) == 0 && a.CheckPluginPathOverride(name) {
		if err := a.RestoreSystemPath(name); err != nil {
			return err
		}
	}

	// 1. 先获取该插件所有已安装的版本
	restoreDetachedOverride := false
	if len(keptCustomSdks) > 0 {
		if err := a.detachPluginPathOverrideFiles(name); err != nil {
			return err
		}
		restoreDetachedOverride = true
		defer func() {
			if err != nil && restoreDetachedOverride {
				if restoreErr := a.HijackSystemPath(name, keptCustomSdks[0].Path); restoreErr != nil {
					err = fmt.Errorf("%w; also failed to restore kept custom SDK path override: %v", err, restoreErr)
				}
			}
		}()
		defer func() {
			if err == nil {
				restoreDetachedOverride = false
			}
		}()
	}

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
	if err != nil {
		return err
	}

	// 5. Delete all associated custom SDKs (None Vfox SDKs)
	if len(keptCustomSdks) > 0 {
		m[name] = keptCustomSdks
	} else if _, ok := m[name]; ok {
		delete(m, name)
	}
	if saveErr := a.saveNonVfoxSdksMap(m); saveErr != nil {
		return saveErr
	}

	if len(keptCustomSdks) > 0 {
		restoreDetachedOverride = false
		if err := a.HijackSystemPath(name, keptCustomSdks[0].Path); err != nil {
			return err
		}
		a.emitEvent("sdk-list-changed")
		return nil
	}

	if vfoxLinkPath := a.getVfoxHomePath("sdks", name); vfoxLinkPath != "" {
		// 6. 保底清理：防止因任何异常导致残留的 SDK 空目录
		_ = os.RemoveAll(vfoxLinkPath)
	}
	return nil
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
			a.emitEvent("system-sdks-ready")
		}
	}

	// Build a clean PATH that excludes .vfox entries, for child processes only.
	// IMPORTANT: Do NOT use os.Setenv here — it modifies global process state
	// and races with concurrent goroutines (RunVfoxCommand, RunVfoxWithProgress, etc.).
	originalPath := os.Getenv("PATH")
	paths := filepath.SplitList(originalPath)
	vfoxSdksDir := a.getVfoxHomePath("sdks")
	var cleanPaths []string
	for _, p := range paths {
		pTrim := strings.TrimSpace(p)
		if pTrim == "" {
			continue
		}
		if vfoxSdksDir != "" && isPathWithin(pTrim, vfoxSdksDir) {
			continue
		}
		cleanPaths = append(cleanPaths, pTrim)
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
	_ = a.writeJSONFile(cacheFile, result)

	a.emitEvent("system-sdks-ready")
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

func (a *App) saveNonVfoxSdksMap(m map[string][]SdkInfo) error {
	return a.writeJSONFile(a.getNonVfoxSdksFile(), m)
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
	if err := validateSDKExecutablePath(exePath); err != nil {
		return err
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}

	if version == "" {
		version = "unknown"
	}

	m := a.GetNonVfoxSdksMap()
	list := m[name]
	for _, existing := range list {
		if strings.EqualFold(filepath.Clean(existing.Path), filepath.Clean(exePath)) {
			return fmt.Errorf("path already exists")
		}
	}
	m[name] = append(m[name], SdkInfo{
		Name:     name,
		Source:   "system",
		Path:     exePath,
		Versions: []SdkVersion{{Version: version}},
	})
	return a.saveNonVfoxSdksMap(m)
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
				sdkLinkPath := a.getVfoxHomePath("sdks", name)
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
	return a.saveNonVfoxSdksMap(m)
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
//  3. {exe_dir}/../../core/     — local `wails build` output in build/bin
//  4. /usr/lib/vfoxg/core/      — Linux DEB/RPM system install
//  5. {cwd}/core/               — dev fallback
func (a *App) getCoreDir() string {
	suffix := filepath.Join(coreOSName(), coreArchName())

	// Build candidate list
	var candidates []string

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		// 1. {exe_dir}/core/ — standard for Windows installer
		candidates = append(candidates, filepath.Join(exeDir, "core"))
		// 2. {exe_dir}/../Resources/core/ — macOS .app bundle
		candidates = append(candidates, filepath.Join(exeDir, "..", "Resources", "core"))
		// 3. {repo}/core/ when running a local binary from build/bin
		candidates = append(candidates, filepath.Join(exeDir, "..", "..", "core"))
	}

	// 4. /usr/lib/vfoxg/core/ — Linux system package
	if stdruntime.GOOS == "linux" {
		candidates = append(candidates, "/usr/lib/vfoxg/core")
	}

	// 5. {cwd}/core/ — dev fallback
	if abs, err := filepath.Abs("core"); err == nil {
		candidates = append(candidates, abs)
	}

	for _, c := range candidates {
		full := filepath.Join(c, suffix)
		exePath := filepath.Join(full, getVfoxExeName())
		if info, err := os.Stat(exePath); err == nil && !info.IsDir() {
			return full
		}
	}

	// Ultimate fallback (may not exist, but avoids empty string)
	baseDir, _ := filepath.Abs("core")
	return filepath.Join(baseDir, suffix)
}

func (a *App) getVfoxExecutable() (string, error) {
	exePath := filepath.Join(a.getCoreDir(), getVfoxExeName())
	if info, err := os.Stat(exePath); err == nil && !info.IsDir() {
		return exePath, nil
	}
	return "", fmt.Errorf("vfox core executable not found at %s; install or bundle core/%s/%s", exePath, coreOSName(), coreArchName())
}

func coreOSName() string {
	osName := stdruntime.GOOS
	if osName == "darwin" {
		return "macos"
	}
	return osName
}

func coreArchName() string {
	switch stdruntime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	default:
		return stdruntime.GOARCH
	}
}

// GetActiveCustomSdk reads the junction target. If it points outside the configured
// VFOX_HOME, it's a Custom SDK and returns the path.
func (a *App) GetActiveCustomSdk(name string) (string, error) {
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if sdkLinkPath == "" {
		return "", fmt.Errorf("unable to resolve vfox home directory")
	}

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
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(sdkLinkPath), target)
			}
			// Targets under VFOX_HOME are managed by vfox, not user-provided SDKs.
			if isPathWithin(target, a.getVfoxHome()) {
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
	a.homeMu.RLock()
	path := strings.TrimSpace(a.vfoxHome)
	a.homeMu.RUnlock()
	if path != "" {
		return path
	}
	if v := strings.TrimSpace(os.Getenv("VFOX_HOME")); v != "" {
		if normalized, err := normalizeDownloadPath(v); err == nil {
			return normalized
		}
		return v
	}
	return a.defaultVfoxHome()
}
