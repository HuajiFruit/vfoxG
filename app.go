package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	homeMu        sync.RWMutex
	vfoxHome      string
	vfoxTaskMutex sync.Mutex
	vfoxTaskBusy  bool
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
		a.emitEvent("vfox-log", "[APP ERROR] "+err.Error())
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		a.emitEvent("vfox-log", "[APP ERROR] "+err.Error())
	}
	go a.ScanSystemSdks()
	go a.RefreshAvailablePlugins() // 后台预热插件市场缓存
}

// getCleanedEnvForVfox returns a copy of the current environment but sanitizes the PATH
// to remove any previously injected vfox SDK/shim paths.
func (a *App) getCleanedEnvForVfox() []string {
	env := os.Environ()
	vfoxHome := strings.TrimSpace(a.getVfoxHome())
	roots := a.vfoxManagedPathRoots()

	for i, e := range env {
		if strings.HasPrefix(strings.ToLower(e), "path=") {
			env[i] = "PATH=" + cleanPathValue(e[5:], roots)
			break
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

func (a *App) vfoxManagedPathRoots() []string {
	var roots []string
	addRoot := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		root = filepath.Clean(root)
		for _, existing := range roots {
			if samePath(existing, root) {
				return
			}
		}
		roots = append(roots, root)
	}

	if vfoxHome := strings.TrimSpace(a.getVfoxHome()); vfoxHome != "" {
		addRoot(filepath.Join(vfoxHome, "cache"))
		addRoot(filepath.Join(vfoxHome, "sdks"))
		addRoot(filepath.Join(vfoxHome, "path-shims"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		legacyHome := filepath.Join(home, ".vfox")
		addRoot(filepath.Join(legacyHome, "cache"))
		addRoot(filepath.Join(legacyHome, "sdks"))
		addRoot(filepath.Join(legacyHome, "path-shims"))
	}
	return roots
}

func cleanPathValue(pathVal string, excludedRoots []string) string {
	parts := filepath.SplitList(pathVal)
	var clean []string
	for _, p := range parts {
		pTrim := strings.TrimSpace(p)
		if pTrim == "" {
			continue
		}
		excluded := false
		for _, root := range excludedRoots {
			if isPathWithin(pTrim, root) {
				excluded = true
				break
			}
		}
		if !excluded {
			clean = append(clean, pTrim)
		}
	}
	return strings.Join(clean, string(filepath.ListSeparator))
}

func (a *App) isVfoxManagedPath(path string) bool {
	for _, root := range a.vfoxManagedPathRoots() {
		if isPathWithin(path, root) {
			return true
		}
	}
	return false
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
	if !isExclusiveVfoxCommand(args) {
		return a.runVfoxCommand(args...)
	}

	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()
	return a.runVfoxCommand(args...)
}

func isExclusiveVfoxCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "add", "install", "remove", "uninstall", "use", "unuse":
		return true
	default:
		return false
	}
}

func (a *App) runVfoxCommand(args ...string) (string, error) {
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

func (a *App) getVersionPathUnlocked(name, version string) (string, error) {
	out, err := a.runVfoxCommand("info", name+"@"+version)
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

func (a *App) getInstalledSdksUnlocked() ([]SdkInfo, error) {
	out, err := a.runVfoxCommand("ls")
	if err != nil {
		return nil, err
	}

	return parseInstalledSdksOutput(out), nil
}

// RunVfoxWithProgress 执行长耗时的 vfox 命令，将输出流实时发送到前端
func (a *App) RunVfoxWithProgress(args []string) error {
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()
	return a.runVfoxWithProgress(args)
}

func (a *App) runVfoxWithProgress(args []string) error {
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

	var outputMu sync.Mutex
	lastOutput := ""
	emitOutputLine := func(line string) {
		cleanLine := ansiRegex.ReplaceAllString(line, "")
		if cleanLine == "" {
			return
		}
		outputMu.Lock()
		lastOutput = cleanLine
		outputMu.Unlock()
		a.emitEvent("vfox-log", cleanLine)
	}

	// 实时读取标准输出
	var readWG sync.WaitGroup
	readWG.Add(2)
	go func() {
		defer readWG.Done()
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
			emitOutputLine(line)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			a.emitEvent("vfox-log", "[STDOUT READ ERROR] "+err.Error())
		}
	}()

	// 实时读取标准错误
	go func() {
		defer readWG.Done()
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
			// Some CLI tools, including vfox during downloads, write progress
			// to stderr even when the command succeeds. Only cmd.Wait should
			// decide whether stderr output means failure.
			emitOutputLine(line)
		}
		if err := scanner.Err(); err != nil && a.ctx != nil {
			a.emitEvent("vfox-log", "[STDERR READ ERROR] "+err.Error())
		}
	}()

	err = cmd.Wait()
	readWG.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		a.emitEvent("vfox-log", "[TIMEOUT] Command cancelled after 30min")
		if err != nil {
			return fmt.Errorf("vfox %v timed out after 30min: %w", args, err)
		}
		return fmt.Errorf("vfox %v timed out after 30min", args)
	}
	if err != nil {
		outputMu.Lock()
		detail := strings.TrimSpace(lastOutput)
		outputMu.Unlock()
		if detail == "" {
			detail = err.Error()
		} else {
			detail = fmt.Sprintf("%s (%v)", detail, err)
		}
		a.emitEvent("vfox-log", "[EXIT ERROR] "+detail)
		return fmt.Errorf("%s", detail)
	}

	a.emitEvent("vfox-log", "[DONE]")
	return nil
}

func (a *App) tryStartVfoxTask() (func(), error) {
	a.vfoxTaskMutex.Lock()
	if a.vfoxTaskBusy {
		a.vfoxTaskMutex.Unlock()
		return nil, fmt.Errorf("another terminal task is already running")
	}
	a.vfoxTaskBusy = true
	a.vfoxTaskMutex.Unlock()

	return func() {
		a.vfoxTaskMutex.Lock()
		a.vfoxTaskBusy = false
		a.vfoxTaskMutex.Unlock()
	}, nil
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

var vfoxHomeMigrationEntries = []string{
	".vfox.toml",
	"cache",
	"plugin",
	"sdks",
}

func hasMigratableVfoxHomeData(path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	for _, entry := range vfoxHomeMigrationEntries {
		entryPath := filepath.Join(path, entry)
		info, err := os.Lstat(entryPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, err
		}
		if info.IsDir() {
			if hasDirectoryEntries(entryPath) {
				return true, nil
			}
			continue
		}
		return true, nil
	}
	return false, nil
}

func hasDirectoryEntries(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true
	}
	return len(entries) > 0
}

func (a *App) migrateVfoxHomeData(from string, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" || samePath(from, to) {
		return nil
	}
	if isPathWithin(to, from) {
		return fmt.Errorf("new download path cannot be inside the current vfox data directory")
	}
	if err := os.MkdirAll(to, 0755); err != nil {
		return err
	}
	for _, entry := range vfoxHomeMigrationEntries {
		src := filepath.Join(from, entry)
		if _, err := os.Lstat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := a.copyPathNoOverwrite(src, filepath.Join(to, entry), from, to); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", entry, err)
		}
	}
	a.emitEvent("vfox-log", "[INFO] Migrated vfox SDK data to "+to)
	return nil
}

func (a *App) copyPathNoOverwrite(src string, dst string, oldRoot string, newRoot string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	} else if !os.IsNotExist(err) {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(src), target)
		}
		target = filepath.Clean(target)
		if isPathWithin(target, oldRoot) {
			rel, err := filepath.Rel(oldRoot, target)
			if err != nil {
				return err
			}
			target = filepath.Join(newRoot, rel)
		}
		return a.ensureJunction(dst, target)
	}
	if info.IsDir() {
		return a.copyDirNoOverwrite(src, dst, info.Mode().Perm(), oldRoot, newRoot)
	}
	return copyFileNoOverwrite(src, dst, info.Mode().Perm())
}

func (a *App) copyDirNoOverwrite(src string, dst string, perm os.FileMode, oldRoot string, newRoot string) error {
	if err := os.MkdirAll(dst, perm); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := a.copyPathNoOverwrite(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), oldRoot, newRoot); err != nil {
			return err
		}
	}
	return os.Chmod(dst, perm)
}

func copyFileNoOverwrite(src string, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}

func (a *App) GetDownloadPathInfo() (DownloadPathInfo, error) {
	path := a.getVfoxHome()
	defaultPath := a.defaultVfoxHome()
	hasMigratableData, err := hasMigratableVfoxHomeData(path)
	if err != nil {
		return DownloadPathInfo{}, err
	}
	return DownloadPathInfo{
		Path:              path,
		DefaultPath:       defaultPath,
		IsDefault:         samePath(path, defaultPath),
		HasMigratableData: hasMigratableData,
	}, nil
}

func (a *App) SetDownloadPath(path string) (DownloadPathInfo, error) {
	return a.SetDownloadPathWithMigration(path, false)
}

func (a *App) SetDownloadPathWithMigration(path string, migrateVfoxData bool) (DownloadPathInfo, error) {
	normalized, err := normalizeDownloadPath(path)
	if err != nil {
		return DownloadPathInfo{}, err
	}
	current := a.getVfoxHome()
	if samePath(current, normalized) {
		return a.GetDownloadPathInfo()
	}
	if migrateVfoxData {
		if err := a.migrateVfoxHomeData(current, normalized); err != nil {
			return DownloadPathInfo{}, err
		}
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
	return a.ResetDownloadPathWithMigration(false)
}

func (a *App) ResetDownloadPathWithMigration(migrateVfoxData bool) (DownloadPathInfo, error) {
	defaultPath := a.defaultVfoxHome()
	current := a.getVfoxHome()
	if samePath(current, defaultPath) {
		return a.GetDownloadPathInfo()
	}
	if migrateVfoxData {
		if err := a.migrateVfoxHomeData(current, defaultPath); err != nil {
			return DownloadPathInfo{}, err
		}
	}
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

type sdkEnvironmentExport struct {
	GeneratedAt  time.Time
	Platform     PlatformInfo
	VfoxInPath   bool
	PathOverride bool
	VfoxSdks     []sdkEnvironmentVfoxSdk
	SystemSdks   []SdkInfo
	CustomSdks   map[string][]SdkInfo
	Warnings     []string
}

type sdkEnvironmentVfoxSdk struct {
	Name             string
	Versions         []SdkVersion
	Detail           SdkDetail
	VersionPaths     map[string]string
	ActiveCustomPath string
}

type SdkEnvironmentImportResult struct {
	Path               string   `json:"path"`
	ImportedCustomSdks int      `json:"importedCustomSdks"`
	SkippedCustomSdks  int      `json:"skippedCustomSdks"`
	VfoxSdksFound      int      `json:"vfoxSdksFound"`
	InstalledVfoxSdks  int      `json:"installedVfoxSdks"`
	SkippedVfoxSdks    int      `json:"skippedVfoxSdks"`
	Warnings           []string `json:"warnings"`
}

// ExportCurrentEnvironmentSdks writes the currently detected SDK environment to a text file.
func (a *App) ExportCurrentEnvironmentSdks() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}

	snapshot := a.collectSdkEnvironmentExport(time.Now())
	defaultDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		defaultDir = home
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export SDK environment",
		DefaultDirectory: defaultDir,
		DefaultFilename:  fmt.Sprintf("vfoxG-sdk-environment-%s.txt", snapshot.GeneratedAt.Format("20060102-150405")),
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".txt") {
		path += ".txt"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(formatSdkEnvironmentExport(snapshot)), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// PreviewCurrentEnvironmentSdks returns the same text that would be exported.
func (a *App) PreviewCurrentEnvironmentSdks() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context is not ready")
	}
	snapshot := a.collectSdkEnvironmentExport(time.Now())
	return formatSdkEnvironmentExport(snapshot), nil
}

func (a *App) ImportSdkEnvironmentFromTxt() (SdkEnvironmentImportResult, error) {
	result := SdkEnvironmentImportResult{}
	if a.ctx == nil {
		return result, fmt.Errorf("application context is not ready")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import SDK environment",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return result, err
	}
	path = strings.TrimSpace(path)
	result.Path = path
	if path == "" {
		return result, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	rows, warnings := parseSdkEnvironmentImport(string(data))
	result.Warnings = append(result.Warnings, warnings...)

	customSdks := a.GetNonVfoxSdksMap()
	type vfoxImportTarget struct {
		Name    string
		Version string
	}
	var vfoxTargets []vfoxImportTarget
	seenVfoxTargets := make(map[string]bool)
	for _, row := range rows {
		switch row.Kind {
		case "vfox":
			name := strings.TrimSpace(row.Name)
			version := normalizeSdkVersion(row.Version)
			if name == "" || isUnknownSdkVersion(version) {
				result.SkippedVfoxSdks++
				continue
			}
			result.VfoxSdksFound++
			key := strings.ToLower(name) + "@" + strings.ToLower(version)
			if !seenVfoxTargets[key] {
				seenVfoxTargets[key] = true
				vfoxTargets = append(vfoxTargets, vfoxImportTarget{Name: name, Version: version})
			}
		case "custom":
			if row.Name == "" || row.Path == "" {
				result.SkippedCustomSdks++
				result.Warnings = append(result.Warnings, "Skipped custom SDK row with missing name or path.")
				continue
			}
			if err := validateSDKExecutablePath(row.Path); err != nil {
				result.SkippedCustomSdks++
				result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s at %s: %v", row.Name, row.Path, err))
				continue
			}
			duplicate := false
			for _, existing := range customSdks[row.Name] {
				if samePath(existing.Path, row.Path) {
					duplicate = true
					break
				}
			}
			if duplicate {
				result.SkippedCustomSdks++
				continue
			}
			version := strings.TrimSpace(row.Version)
			if isUnknownSdkVersion(version) {
				version = a.DetectSdkPathVersion(row.Name, row.Path)
			}
			if version == "" {
				version = "unknown"
			}
			customSdks[row.Name] = append(customSdks[row.Name], SdkInfo{
				Name:     row.Name,
				Source:   "system",
				Path:     row.Path,
				Versions: []SdkVersion{{Version: version}},
			})
			result.ImportedCustomSdks++
		}
	}

	if result.ImportedCustomSdks > 0 {
		if err := a.saveNonVfoxSdksMap(customSdks); err != nil {
			return result, err
		}
		a.emitEvent("sdk-list-changed")
	}
	if len(vfoxTargets) > 0 {
		releaseTask, err := a.tryStartVfoxTask()
		if err != nil {
			a.emitEvent("vfox-busy")
			return result, err
		}
		defer releaseTask()

		addedPlugins := make(map[string]bool)
		if plugins, err := a.GetAddedPlugins(); err == nil {
			for _, plugin := range plugins {
				addedPlugins[strings.ToLower(strings.TrimSpace(plugin))] = true
			}
		} else {
			result.Warnings = append(result.Warnings, "Unable to list added vfox plugins: "+err.Error())
		}
		for _, target := range vfoxTargets {
			pluginKey := strings.ToLower(target.Name)
			if !addedPlugins[pluginKey] {
				if err := a.runVfoxWithProgress([]string{"add", target.Name}); err != nil {
					result.SkippedVfoxSdks++
					result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s@%s: failed to add vfox plugin: %v", target.Name, target.Version, err))
					continue
				}
				addedPlugins[pluginKey] = true
			}
			if err := a.installVersionUnlocked(target.Name, target.Version); err != nil {
				result.SkippedVfoxSdks++
				result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s@%s: failed to install vfox SDK: %v", target.Name, target.Version, err))
				continue
			}
			result.InstalledVfoxSdks++
		}
		a.emitEvent("sdk-list-changed")
	}
	return result, nil
}

func (a *App) collectSdkEnvironmentExport(generatedAt time.Time) sdkEnvironmentExport {
	snapshot := sdkEnvironmentExport{
		GeneratedAt: generatedAt,
		Platform:    a.GetPlatformInfo(),
		CustomSdks:  a.GetNonVfoxSdksMap(),
	}

	if inPath, err := a.CheckVfoxInPath(); err == nil {
		snapshot.VfoxInPath = inPath
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Unable to check vfox PATH status: "+err.Error())
	}
	snapshot.PathOverride = a.CheckAnyPathOverride()

	if vfoxSdks, err := a.GetInstalledSdks(); err == nil {
		sort.Slice(vfoxSdks, func(i, j int) bool { return vfoxSdks[i].Name < vfoxSdks[j].Name })
		for _, sdk := range vfoxSdks {
			exportSdk := sdkEnvironmentVfoxSdk{
				Name:         sdk.Name,
				Versions:     sdk.Versions,
				VersionPaths: make(map[string]string),
				Detail:       SdkDetail{Name: sdk.Name},
			}
			if detail, err := a.GetSdkDetail(sdk.Name); err == nil {
				exportSdk.Detail = detail
				for _, version := range detail.Versions {
					if path, err := a.GetVersionPath(sdk.Name, version.Version); err == nil {
						exportSdk.VersionPaths[version.Version] = path
					}
				}
			} else {
				snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Unable to load vfox SDK detail for %s: %v", sdk.Name, err))
			}
			if activeCustomPath, err := a.GetActiveCustomSdk(sdk.Name); err == nil {
				exportSdk.ActiveCustomPath = activeCustomPath
			}
			snapshot.VfoxSdks = append(snapshot.VfoxSdks, exportSdk)
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Unable to load vfox SDK list: "+err.Error())
	}

	a.ScanSystemSdks()
	snapshot.SystemSdks = a.GetCachedSystemSdks()
	sort.Slice(snapshot.SystemSdks, func(i, j int) bool { return snapshot.SystemSdks[i].Name < snapshot.SystemSdks[j].Name })

	return snapshot
}

func formatSdkEnvironmentExport(snapshot sdkEnvironmentExport) string {
	var b strings.Builder
	writeLine := func(format string, args ...interface{}) {
		if len(args) == 0 {
			b.WriteString(format)
		} else {
			b.WriteString(fmt.Sprintf(format, args...))
		}
		b.WriteString("\r\n")
	}

	writeLine("vfoxG SDK Environment Export")
	writeLine("Generated: %s", snapshot.GeneratedAt.Format(time.RFC3339))
	writeLine("")
	writeLine("Platform")
	writeLine("  OS: %s", emptyFallback(snapshot.Platform.Name, snapshot.Platform.OS))
	writeLine("  Core: %s/%s", snapshot.Platform.CoreOS, snapshot.Platform.CoreArch)
	writeLine("  VFOX_HOME: %s", emptyFallback(snapshot.Platform.DownloadPath, "(empty)"))
	writeLine("  Default VFOX_HOME: %s", emptyFallback(snapshot.Platform.DefaultDownloadPath, "(empty)"))
	writeLine("  vfox in PATH: %s", yesNo(snapshot.VfoxInPath))
	writeLine("  SDK PATH override active: %s", yesNo(snapshot.PathOverride))
	writeLine("")

	if len(snapshot.Warnings) > 0 {
		writeLine("Warnings")
		for _, warning := range snapshot.Warnings {
			writeLine("  - %s", warning)
		}
		writeLine("")
	}

	writeLine("Vfox SDKs")
	writeLine("Name | Version | Current | Path")
	writeLine("--- | --- | --- | ---")
	if len(snapshot.VfoxSdks) == 0 {
		writeLine("(none) |  |  | ")
	} else {
		for _, sdk := range snapshot.VfoxSdks {
			details := sdk.Detail.Versions
			if len(details) == 0 {
				for _, version := range sdk.Versions {
					details = append(details, SdkVersionDetail{
						Version:   version.Version,
						IsCurrent: version.Version != "" && version.Version == sdk.Detail.Current,
					})
				}
			}
			if len(details) == 0 {
				writeLine("%s | (none) | no | ", tableCell(sdk.Name))
			}
			for _, version := range details {
				current := "no"
				if version.IsCurrent {
					current = "yes"
				}
				path := strings.TrimSpace(sdk.VersionPaths[version.Version])
				if path == "" && version.IsCurrent && sdk.ActiveCustomPath != "" {
					path = sdk.ActiveCustomPath
				}
				writeLine("%s | %s | %s | %s", tableCell(sdk.Name), tableCell(version.Version), current, tableCell(path))
			}
		}
	}
	writeLine("")

	writeLine("Custom SDKs")
	writeCustomSdks(writeLine, snapshot.CustomSdks)
	writeLine("")

	writeLine("System SDKs")
	writeLine("Name | Version | Executable")
	writeLine("--- | --- | ---")
	if len(snapshot.SystemSdks) == 0 {
		writeLine("(none) |  | ")
	} else {
		for _, sdk := range snapshot.SystemSdks {
			if len(sdk.Versions) == 0 {
				writeLine("%s | (unknown) | %s", tableCell(sdk.Name), tableCell(sdk.Path))
			}
			for _, version := range sdk.Versions {
				writeLine("%s | %s | %s", tableCell(sdk.Name), tableCell(emptyFallback(version.Version, "(unknown)")), tableCell(sdk.Path))
			}
		}
	}
	return b.String()
}

func writeCustomSdks(writeLine func(string, ...interface{}), customSdks map[string][]SdkInfo) {
	writeLine("Name | Version | Path")
	writeLine("--- | --- | ---")
	if len(customSdks) == 0 {
		writeLine("(none) |  | ")
		return
	}
	names := make([]string, 0, len(customSdks))
	for name := range customSdks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		list := append([]SdkInfo(nil), customSdks[name]...)
		sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
		if len(list) == 0 {
			writeLine("%s | (none) | ", tableCell(name))
			continue
		}
		for _, sdk := range list {
			if len(sdk.Versions) == 0 {
				writeLine("%s | (unknown) | %s", tableCell(name), tableCell(sdk.Path))
				continue
			}
			for _, version := range sdk.Versions {
				writeLine("%s | %s | %s", tableCell(name), tableCell(emptyFallback(version.Version, "(unknown)")), tableCell(sdk.Path))
			}
		}
	}
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "/")
	return strings.TrimSpace(value)
}

func isUnknownSdkVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version == "" || strings.EqualFold(version, "unknown") || strings.EqualFold(version, "(unknown)")
}

type sdkEnvironmentImportRow struct {
	Kind    string
	Name    string
	Version string
	Path    string
	Current bool
}

func parseSdkEnvironmentImport(data string) ([]sdkEnvironmentImportRow, []string) {
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	section := ""
	currentCustomName := ""
	var rows []sdkEnvironmentImportRow
	var warnings []string

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		switch lower {
		case "vfox sdks":
			section = "vfox"
			currentCustomName = ""
			continue
		case "custom sdks", "custom sdk paths":
			section = "custom"
			currentCustomName = ""
			continue
		case "system sdks":
			section = "system"
			currentCustomName = ""
			continue
		}

		if parts, ok := parseExportTableLine(line); ok {
			if len(parts) == 0 || isExportTableHeader(parts) {
				continue
			}
			switch section {
			case "vfox":
				if len(parts) >= 2 && !strings.EqualFold(parts[0], "(none)") {
					rows = append(rows, sdkEnvironmentImportRow{
						Kind:    "vfox",
						Name:    parts[0],
						Version: parts[1],
						Current: len(parts) >= 3 && strings.EqualFold(parts[2], "yes"),
						Path:    tablePart(parts, 3),
					})
				}
			case "custom":
				if len(parts) >= 3 && !strings.EqualFold(parts[0], "(none)") {
					rows = append(rows, sdkEnvironmentImportRow{
						Kind:    "custom",
						Name:    parts[0],
						Version: parts[1],
						Path:    parts[2],
					})
				}
			}
			continue
		}

		// Backward compatibility with older exports:
		// Custom SDK Paths / "  golang" / "    - Path: ..." / "      Version: ..."
		if section == "custom" {
			if strings.HasPrefix(rawLine, "  ") && !strings.HasPrefix(rawLine, "    ") {
				currentCustomName = line
				continue
			}
			if strings.HasPrefix(line, "- Path:") && currentCustomName != "" {
				rows = append(rows, sdkEnvironmentImportRow{
					Kind: "custom",
					Name: currentCustomName,
					Path: strings.TrimSpace(strings.TrimPrefix(line, "- Path:")),
				})
				continue
			}
			if strings.HasPrefix(line, "Version:") && len(rows) > 0 && rows[len(rows)-1].Kind == "custom" && rows[len(rows)-1].Version == "" {
				rows[len(rows)-1].Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
				continue
			}
		}

		if section == "vfox" {
			if strings.HasPrefix(rawLine, "  ") && !strings.HasPrefix(rawLine, "    ") {
				currentCustomName = line
				continue
			}
			if strings.HasPrefix(line, "- ") && currentCustomName != "" {
				version := strings.TrimSpace(strings.TrimPrefix(line, "- "))
				version = strings.TrimSuffix(version, " (current)")
				rows = append(rows, sdkEnvironmentImportRow{
					Kind:    "vfox",
					Name:    currentCustomName,
					Version: strings.TrimSpace(version),
					Current: strings.Contains(line, "(current)"),
				})
				continue
			}
		}
	}

	if len(rows) == 0 {
		warnings = append(warnings, "No SDK rows were found in the selected text file.")
	}
	return rows, warnings
}

func parseExportTableLine(line string) ([]string, bool) {
	if !strings.Contains(line, "|") {
		return nil, false
	}
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

func isExportTableHeader(parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	first := strings.TrimSpace(parts[0])
	if strings.EqualFold(first, "Name") {
		return true
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Trim(part, "- ") != "" {
			return false
		}
	}
	return true
}

func tablePart(parts []string, index int) string {
	if index >= len(parts) {
		return ""
	}
	return parts[index]
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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
	Path              string `json:"path"`
	DefaultPath       string `json:"defaultPath"`
	IsDefault         bool   `json:"isDefault"`
	HasMigratableData bool   `json:"hasMigratableData"`
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
	return a.getInstalledSdksUnlocked()
}

func parseInstalledSdksOutput(out string) []SdkInfo {
	lines := strings.Split(out, "\n")
	sdks := make([]SdkInfo, 0)
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

	return sdks
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

			isOfficial := isOfficialPluginStatus(status)
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

func isOfficialPluginStatus(status string) bool {
	status = strings.TrimSpace(status)
	return status == "✓" || status == "√" || strings.EqualFold(status, "true") || strings.EqualFold(status, "yes")
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
	activeCustomPath, _ := a.GetActiveCustomSdk(name)
	hasActiveCustom := activeCustomPath != ""
	currentOut, _ := a.RunVfoxCommand("current", name)
	currentVer := parseCurrentSdkVersion(name, currentOut)
	if hasActiveCustom {
		currentVer = ""
	}

	// Fallback: if vfox current fails, read from .vfox.toml directly only
	// when there is no custom SDK junction active for this plugin.
	allowConfigFallback := true
	if currentVer == "" || strings.Contains(currentVer, "no current") {
		if hasActiveCustom {
			allowConfigFallback = false
		}
	}
	if allowConfigFallback && (currentVer == "" || strings.Contains(currentVer, "no current")) {
		if data, err := os.ReadFile(a.getVfoxHomePath(".vfox.toml")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, name+" ") || strings.HasPrefix(trimmed, name+"=") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						v := strings.TrimSpace(parts[1])
						v = strings.Trim(v, "\"")
						currentVer = normalizeSdkVersion(v)
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

	detail := parseSdkDetailOutput(name, currentVer, out)
	if hasActiveCustom {
		detail.Current = ""
		for i := range detail.Versions {
			detail.Versions[i].IsCurrent = false
		}
	}
	return detail, nil
}

func parseSdkDetailOutput(name string, currentVer string, out string) SdkDetail {
	currentVer = normalizeSdkVersion(currentVer)
	lines := strings.Split(out, "\n")
	detail := SdkDetail{Name: name, Current: currentVer}
	markerCurrentCount := 0

	for _, line := range lines {
		ver, markerCurrent, ok := parseSdkDetailVersionLine(line)
		if !ok {
			continue
		}

		isCurrent := false
		if currentVer != "" {
			isCurrent = sameSdkVersion(ver, currentVer)
		} else if markerCurrent {
			isCurrent = true
			detail.Current = ver
			markerCurrentCount++
		}

		detail.Versions = append(detail.Versions, SdkVersionDetail{
			Version:   ver,
			IsCurrent: isCurrent,
		})
	}

	// Some vfox outputs can leave more than one line marked current. Do not expose
	// multiple active versions to the UI when `vfox current` did not confirm one.
	if currentVer == "" && markerCurrentCount > 1 {
		detail.Current = ""
		for i := range detail.Versions {
			detail.Versions[i].IsCurrent = false
		}
	}

	return detail
}

func parseCurrentSdkVersion(name string, out string) string {
	name = strings.TrimSpace(name)
	for _, rawLine := range strings.Split(out, "\n") {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "no current") ||
			strings.Contains(lower, "not installed") ||
			strings.Contains(lower, "not supported") {
			continue
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, "->"))
		for _, marker := range []string{"<-- current", "<— current", "<- current"} {
			if strings.HasSuffix(line, marker) {
				line = strings.TrimSpace(strings.TrimSuffix(line, marker))
				break
			}
		}

		if name != "" {
			lowerLine := strings.ToLower(line)
			lowerName := strings.ToLower(name)
			switch {
			case strings.HasPrefix(lowerLine, lowerName+"@"):
				line = strings.TrimSpace(line[len(name)+1:])
			case strings.HasPrefix(lowerLine, lowerName+" ->"):
				_, rest, _ := strings.Cut(line, "->")
				line = strings.TrimSpace(rest)
			case strings.HasPrefix(lowerLine, lowerName+":"):
				line = strings.TrimSpace(line[len(name)+1:])
			case strings.HasPrefix(lowerLine, lowerName+" "):
				line = strings.TrimSpace(line[len(name):])
			}
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, "@"))
		if fields := strings.Fields(line); len(fields) > 0 {
			line = fields[0]
		}
		line = strings.Trim(line, "\"'")
		if version := normalizeSdkVersion(line); version != "" {
			return version
		}
	}
	return ""
}

func normalizeSdkVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}

func sameSdkVersion(a string, b string) bool {
	return strings.EqualFold(normalizeSdkVersion(a), normalizeSdkVersion(b))
}

func parseSdkDetailVersionLine(line string) (version string, isCurrent bool, ok bool) {
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	if line == "" {
		return "", false, false
	}

	if strings.HasPrefix(line, "->") {
		isCurrent = true
		line = strings.TrimSpace(strings.TrimPrefix(line, "->"))
	}

	for _, marker := range []string{"<-- current", "<— current", "<- current"} {
		if strings.HasSuffix(line, marker) {
			isCurrent = true
			line = strings.TrimSpace(strings.TrimSuffix(line, marker))
			break
		}
	}

	if line == "" || strings.Contains(line, "installed sdk") || strings.HasPrefix(line, "custom-sys-") {
		return "", false, false
	}
	return line, isCurrent, true
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// UseVersion 切换到指定版本
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

func (a *App) useVersionUnlocked(name, version string) (string, error) {
	if activeCustomPath, err := a.GetActiveCustomSdk(name); err == nil && activeCustomPath != "" {
		a.removeJunctionIfExists(a.getVfoxHomePath("sdks", name))
	}

	// 使用 RunVfoxCommand 正常等待 vfox use 完成，因为我们已经在环境变量里伪装成了 cmd。
	if _, err := a.runVfoxCommand("use", "--global", name+"@"+version); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	runtimeRoot, err := a.resolveVersionRuntimeRootUnlocked(name, version)
	if err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, runtimeRoot); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}

	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")
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

func (a *App) resolveVersionRuntimeRoot(name string, version string) (string, error) {
	versionPath, err := a.GetVersionPath(name, version)
	if err != nil {
		return "", err
	}
	return a.resolveVersionRuntimeRootFromPath(name, version, versionPath)
}

func (a *App) resolveVersionRuntimeRootUnlocked(name string, version string) (string, error) {
	versionPath, err := a.getVersionPathUnlocked(name, version)
	if err != nil {
		return "", err
	}
	return a.resolveVersionRuntimeRootFromPath(name, version, versionPath)
}

func (a *App) resolveVersionRuntimeRootFromPath(name string, version string, versionPath string) (string, error) {
	versionPath = strings.TrimSpace(versionPath)
	if versionPath == "" {
		return "", fmt.Errorf("unable to resolve install path for %s@%s", name, version)
	}
	if sdkRootHasExecutable(versionPath, name) {
		return versionPath, nil
	}

	entries, err := os.ReadDir(versionPath)
	if err != nil {
		return versionPath, nil
	}
	var singleDir string
	dirCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirCount++
		childPath := filepath.Join(versionPath, entry.Name())
		if singleDir == "" {
			singleDir = childPath
		}
		if sdkRootHasExecutable(childPath, name) {
			return childPath, nil
		}
	}
	if dirCount == 1 && singleDir != "" {
		return singleDir, nil
	}
	return versionPath, nil
}

func sdkRootHasExecutable(root string, name string) bool {
	for _, dir := range []string{"", "bin", "Scripts", "sbin"} {
		baseDir := root
		if dir != "" {
			baseDir = filepath.Join(root, dir)
		}
		for _, exe := range sdkExecutableAliases(name) {
			for _, candidate := range executableFileCandidates(exe) {
				if isRegularFile(filepath.Join(baseDir, candidate)) {
					return true
				}
			}
		}
	}
	return false
}

func sdkExecutableAliases(name string) []string {
	aliases := []string{strings.TrimSpace(name)}
	for _, def := range systemSDKDefs {
		if def.Name == name {
			aliases = append(aliases, def.Exe)
			break
		}
	}
	switch strings.ToLower(name) {
	case "python":
		aliases = append(aliases, "python3")
	case "nodejs":
		aliases = append(aliases, "node", "npm", "npx")
	case "golang":
		aliases = append(aliases, "go")
	}
	return uniqueNonEmptyStrings(aliases)
}

func executableFileCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if stdruntime.GOOS != "windows" || filepath.Ext(name) != "" {
		return []string{name}
	}
	return []string{name + ".exe", name + ".cmd", name + ".bat", name}
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

// removeJunctionIfExists removes a junction/directory if it exists.
func (a *App) removeJunctionIfExists(linkPath string) {
	linkPath = strings.TrimSpace(linkPath)
	if linkPath == "" {
		return
	}
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
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	if err := validateSDKExecutablePath(exePath); err != nil {
		return "", err
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return "", err
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()

	return a.useCustomSdkUnlocked(name, exePath)
}

func (a *App) useCustomSdkUnlocked(name string, exePath string) (string, error) {
	root := a.getSdkRoot(exePath)

	// 1. Clear the vfox selection so that both don't show "当前" at the same time.
	// We MUST do this synchronously BEFORE overwriting the symlink, otherwise vfox might delete our custom symlink.
	if err := a.clearGlobalSdkSelectionUnlocked(name); err != nil {
		return "", err
	}

	// 2. Create junction: ~/.vfox/sdks/{name} -> system SDK root
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, root); err != nil {
		return "", fmt.Errorf("failed to create SDK junction: %v", err)
	}

	a.emitEvent("vfox-log", fmt.Sprintf("Activating %s (system)...", name))
	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")

	return "ok", nil
}

// UnuseVersion 取消当前 SDK 的全局版本设置，并清理 GUI 管理的链接
func (a *App) UnuseVersion(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()

	return a.unuseVersionUnlocked(name)
}

func (a *App) unuseVersionUnlocked(name string) (string, error) {
	if err := a.clearGlobalSdkSelectionUnlocked(name); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	a.removeJunctionIfExists(a.getVfoxHomePath("sdks", name))
	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")
	return "ok", nil
}

func (a *App) clearGlobalSdkSelection(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	return a.clearGlobalSdkSelectionUnlocked(name)
}

func (a *App) clearGlobalSdkSelectionUnlocked(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	var cmdErr error
	if _, err := a.runVfoxCommand("unuse", "--global", name); err != nil {
		cmdErr = err
	}
	if err := a.removeGlobalSdkSelectionFromConfig(name); err != nil {
		return err
	}
	if cmdErr != nil && !isBenignUnuseError(cmdErr) {
		return cmdErr
	}
	return nil
}

func isBenignUnuseError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no current") ||
		strings.Contains(message, "not installed") ||
		strings.Contains(message, "not supported")
}

func (a *App) removeGlobalSdkSelectionFromConfig(name string) error {
	configPath := a.getVfoxHomePath(".vfox.toml")
	if strings.TrimSpace(configPath) == "" {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updated, changed := removeSdkSelectionFromVfoxToml(string(data), name)
	if !changed {
		return nil
	}
	return os.WriteFile(configPath, []byte(updated), 0644)
}

func removeSdkSelectionFromVfoxToml(data string, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return data, false
	}
	lines := strings.SplitAfter(data, "\n")
	var b strings.Builder
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if isSdkSelectionConfigLine(trimmed, name) {
			changed = true
			continue
		}
		b.WriteString(line)
	}
	return b.String(), changed
}

func isSdkSelectionConfigLine(line string, name string) bool {
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}
	return strings.HasPrefix(line, name+" ") || strings.HasPrefix(line, name+"=")
}

// InstallVersion 安装指定版本 (会耗时，并且会产生 vfox-log 进度事件)
func (a *App) InstallVersion(name, version string) error {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return fmt.Errorf("plugin name and version cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	return a.installVersionUnlocked(name, version)
}

func (a *App) installVersionUnlocked(name, version string) error {
	return a.runVfoxWithProgress([]string{"install", "-y", name + "@" + version})
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
	releaseTask, lockErr := a.tryStartVfoxTask()
	if lockErr != nil {
		a.emitEvent("vfox-busy")
		return lockErr
	}
	defer releaseTask()

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

	sdks, err := a.getInstalledSdksUnlocked()
	if err == nil {
		for _, sdk := range sdks {
			if sdk.Name == name && sdk.Source == "vfox" {
				// 2. 逐个卸载版本 (vfox uninstall 会顺带清理它建立的链接和环境配置)
				for _, v := range sdk.Versions {
					_ = a.runVfoxWithProgress([]string{"uninstall", name + "@" + v.Version})
				}
				break
			}
		}
	}

	// 3. 执行 vfox unuse 以确保没有任何全局/局部环境残留 (防范于未然，忽略错误)
	_, _ = a.runVfoxCommand("unuse", "-g", name)
	_, _ = a.runVfoxCommand("unuse", "-p", name)
	_, _ = a.runVfoxCommand("unuse", "-s", name)

	// 4. 彻底删除插件
	err = a.runVfoxWithProgress([]string{"remove", "-y", name})
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
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return fmt.Errorf("plugin name and version cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	return a.uninstallVersionUnlocked(name, version)
}

func (a *App) uninstallVersionUnlocked(name, version string) error {
	return a.runVfoxWithProgress([]string{"uninstall", name + "@" + version})
}

// GetVersionPath 获取指定版本的 SDK 绝对安装路径
func (a *App) GetVersionPath(name, version string) (string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		return "", fmt.Errorf("plugin name and version cannot be empty")
	}
	return a.getVersionPathUnlocked(name, version)
}

// SearchVersions 搜索 SDK 的可用版本，网络错误时返回空列表
func (a *App) SearchVersions(name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{}, fmt.Errorf("plugin name cannot be empty")
	}
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
			cachedResult = a.filterSystemSdks(cachedResult)
			systemSdkCacheMu.Lock()
			systemSdkCache = cachedResult
			systemSdkCacheMu.Unlock()
			if len(cachedResult) > 0 {
				a.emitEvent("system-sdks-ready")
			}
		}
	}

	// Build a clean PATH that excludes vfox-managed entries, for child processes only.
	// IMPORTANT: Do NOT use os.Setenv here — it modifies global process state
	// and races with concurrent goroutines (RunVfoxCommand, RunVfoxWithProgress, etc.).
	originalPath := os.Getenv("PATH")
	cleanPathStr := cleanPathValue(originalPath, a.vfoxManagedPathRoots())

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
	result := make([]SdkInfo, 0, len(systemSDKDefs))

	for _, def := range systemSDKDefs {
		wg.Add(1)
		go func(d systemSDKDef) {
			defer wg.Done()
			for _, exePath := range findExecutableCandidates(d.Exe, cleanEnv) {
				if exePath == "" || a.isVfoxManagedPath(exePath) {
					continue
				}
				ver := a.tryGetVersionWithEnv(exePath, d.VerArgs, cleanEnv)
				if ver == "" || !isUsableSystemVersion(ver) {
					continue
				}
				mu.Lock()
				result = append(result, SdkInfo{
					Name:     d.Name,
					Source:   "system",
					Path:     exePath,
					Versions: []SdkVersion{{Version: ver}},
				})
				mu.Unlock()
				return
			}
		}(def)
	}
	wg.Wait()

	result = a.filterSystemSdks(result)

	systemSdkCacheMu.Lock()
	systemSdkCache = result
	systemSdkCacheMu.Unlock()

	// Write updated results to cache file
	_ = a.writeJSONFile(cacheFile, result)

	a.emitEvent("system-sdks-ready")
}

func (a *App) filterSystemSdks(sdks []SdkInfo) []SdkInfo {
	result := make([]SdkInfo, 0, len(sdks))
	for _, sdk := range sdks {
		if a.isVfoxManagedPath(sdk.Path) {
			continue
		}
		versions := make([]SdkVersion, 0, len(sdk.Versions))
		for _, version := range sdk.Versions {
			if isUsableSystemVersion(version.Version) {
				versions = append(versions, version)
			}
		}
		if len(versions) == 0 {
			continue
		}
		sdk.Versions = versions
		result = append(result, sdk)
	}
	return result
}

func isUsableSystemVersion(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" || strings.EqualFold(version, "unknown") {
		return false
	}
	lower := strings.ToLower(version)
	badFragments := []string{
		"vfoxg:",
		"not available under",
		"not found",
		"not recognized",
		"run without arguments to install",
		"access is denied",
	}
	for _, fragment := range badFragments {
		if strings.Contains(lower, fragment) {
			return false
		}
	}
	return true
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
	name = strings.TrimSpace(name)
	exePath = strings.TrimSpace(exePath)
	if name == "" || exePath == "" {
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
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
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
		if samePath(existing.Path, exePath) {
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
	name = strings.TrimSpace(name)
	exePath = strings.TrimSpace(exePath)
	if name == "" || exePath == "" {
		return fmt.Errorf("plugin name and path cannot be empty")
	}
	m := a.GetNonVfoxSdksMap()
	list := m[name]
	var newList []SdkInfo
	for _, existing := range list {
		if !samePath(existing.Path, exePath) {
			newList = append(newList, existing)
		} else {
			// Only remove the sdk symlink if it currently points to the SDK being removed
			activePath, _ := a.GetActiveCustomSdk(name)
			if samePath(activePath, exePath) {
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
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
	exe = normalizeExecutableName(exe)

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
func normalizeExecutableName(exe string) string {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(exe)))
	for _, suffix := range []string{".exe", ".cmd", ".bat", ".com"} {
		name = strings.TrimSuffix(name, suffix)
	}
	switch name {
	case "pythonw", "python3", "python3w":
		return "python"
	case "nodejs":
		return "node"
	}
	return name
}

func (a *App) GetAllSdks() ([]SdkInfo, error) {
	vfoxSdks, err := a.GetInstalledSdks()
	if err != nil {
		vfoxSdks = []SdkInfo{}
	}

	cached := a.GetCachedSystemSdks()

	seen := make(map[string]bool)
	result := make([]SdkInfo, 0, len(vfoxSdks)+len(cached))

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
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
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
			// Targets under any managed vfox root are not user-provided SDKs.
			if a.isVfoxManagedPath(target) {
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
