//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	vfoxPathMarkerLabel     = "vfoxG PATH"
	vfoxSDKPathMarkerPrefix = "vfoxG SDK PATH "
)

func hideWindow(cmd *exec.Cmd) {
	// Not needed on Unix
}

func getVfoxExeName() string {
	return "vfox"
}

func (a *App) ensureJunction(linkPath string, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			existing, readErr := os.Readlink(linkPath)
			if readErr == nil && existing == target {
				return nil
			}
		} else if fi.IsDir() {
			if linkPath == target {
				return nil
			}
		}
	}
	_ = os.RemoveAll(linkPath)
	_ = os.MkdirAll(filepath.Dir(linkPath), 0755)

	return os.Symlink(target, linkPath)
}

func (a *App) CheckVfoxInPath() (bool, error) {
	coreDir := filepath.Clean(a.getCoreDir())
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(strings.TrimSpace(entry)) == coreDir {
			return true, nil
		}
	}
	if unixManagedBlockExists(vfoxPathMarkerLabel) {
		return true, nil
	}
	return false, nil
}

func (a *App) AddVfoxToPath() error {
	coreDir, err := a.getVfoxExecutable()
	if err != nil {
		return err
	}
	return unixWritePathBlock(vfoxPathMarkerLabel, []string{filepath.Dir(coreDir)})
}

func (a *App) RemoveVfoxFromPath() error {
	return unixRemoveManagedBlock(vfoxPathMarkerLabel)
}

func (a *App) HijackSystemPath(name string, exePath string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}
	if strings.TrimSpace(exePath) != "" {
		if err := validateSDKExecutablePath(exePath); err != nil {
			return err
		}
		sdkRoot := a.getSdkRoot(exePath)
		sdkLinkPath := a.getVfoxHomePath("sdks", name)
		if err := a.ensureJunction(sdkLinkPath, sdkRoot); err != nil {
			return err
		}
	}

	sdkPath := a.getVfoxHomePath("sdks", name)
	return unixWritePathBlock(unixSDKMarkerLabel(name), unixSDKPathEntries(sdkPath))
}

func (a *App) RestoreSystemPath(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	return unixRemoveManagedBlock(unixSDKMarkerLabel(name))
}

func (a *App) detachPluginPathOverrideFiles(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if sdkLinkPath == "" {
		return fmt.Errorf("unable to resolve vfox home directory")
	}
	a.removeJunctionIfExists(sdkLinkPath)
	return nil
}

func (a *App) HijackPluginSystemPath(pluginName string) error {
	m := a.GetNonVfoxSdksMap()
	if list, ok := m[pluginName]; ok && len(list) > 0 {
		return a.HijackSystemPath(pluginName, list[0].Path)
	}
	return a.HijackSystemPath(pluginName, "")
}

func (a *App) RestorePluginSystemPath(pluginName string) error {
	return a.RestoreSystemPath(pluginName)
}

func (a *App) CheckPluginWin11CompatMode(pluginName string) bool {
	return unixManagedBlockExists(unixSDKMarkerLabel(pluginName))
}

func (a *App) CheckWin11CompatMode() bool {
	for _, profile := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profile)
		if err == nil && strings.Contains(string(data), "# >>> "+vfoxSDKPathMarkerPrefix) {
			return true
		}
	}
	return false
}

func findExecutable(exe string, cleanEnv []string) string {
	if strings.ContainsRune(exe, filepath.Separator) {
		if isExecutableFile(exe) {
			return exe
		}
		return ""
	}

	pathValue := os.Getenv("PATH")
	for _, e := range cleanEnv {
		if strings.HasPrefix(e, "PATH=") {
			pathValue = strings.TrimPrefix(e, "PATH=")
			break
		}
	}

	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, exe)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	return ""
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0111 != 0
}

func unixSDKMarkerLabel(pluginName string) string {
	return vfoxSDKPathMarkerPrefix + strings.TrimSpace(pluginName)
}

func unixSDKPathEntries(sdkPath string) []string {
	return []string{
		sdkPath,
		filepath.Join(sdkPath, "bin"),
		filepath.Join(sdkPath, "sbin"),
	}
}

func unixWritePathBlock(label string, paths []string) error {
	profilePath, err := unixShellProfilePath()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no PATH entries to write")
	}

	block := unixManagedPathBlock(label, paths)
	data, _ := os.ReadFile(profilePath)
	updated := unixRemoveManagedBlockFromString(string(data), label)
	updated = strings.TrimRight(updated, "\r\n")
	if updated != "" {
		updated += "\n\n"
	}
	updated += block
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(profilePath, []byte(updated), 0644)
}

func unixRemoveManagedBlock(label string) error {
	var lastErr error
	changed := false
	for _, profilePath := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		updated := unixRemoveManagedBlockFromString(string(data), label)
		if updated == string(data) {
			continue
		}
		if err := os.WriteFile(profilePath, []byte(updated), 0644); err != nil {
			lastErr = err
			continue
		}
		changed = true
	}
	if lastErr != nil {
		return lastErr
	}
	if !changed {
		return nil
	}
	return nil
}

func unixManagedBlockExists(label string) bool {
	start, _ := unixManagedBlockMarkers(label)
	for _, profilePath := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profilePath)
		if err == nil && strings.Contains(string(data), start) {
			return true
		}
	}
	return false
}

func unixManagedPathBlock(label string, paths []string) string {
	start, end := unixManagedBlockMarkers(label)
	var quoted []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		quoted = append(quoted, shellQuote(filepath.Clean(p)))
	}
	if len(quoted) == 0 {
		quoted = append(quoted, `"$PATH"`)
	} else {
		quoted = append(quoted, `"$PATH"`)
	}

	return strings.Join([]string{
		start,
		"# Added by vfoxG. Remove this block from vfoxG settings.",
		"export PATH=" + strings.Join(quoted, ":"),
		end,
		"",
	}, "\n")
}

func unixManagedBlockMarkers(label string) (string, string) {
	cleanLabel := strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(label))
	return "# >>> " + cleanLabel + " >>>", "# <<< " + cleanLabel + " <<<"
}

func unixRemoveManagedBlockFromString(data string, label string) string {
	start, end := unixManagedBlockMarkers(label)
	for {
		startIdx := strings.Index(data, start)
		if startIdx < 0 {
			return data
		}
		endIdx := strings.Index(data[startIdx:], end)
		if endIdx < 0 {
			return data
		}
		endIdx = startIdx + endIdx + len(end)
		for endIdx < len(data) && (data[endIdx] == '\n' || data[endIdx] == '\r') {
			endIdx++
		}
		data = strings.TrimRight(data[:startIdx], "\r\n") + "\n" + data[endIdx:]
		data = strings.TrimLeft(data, "\r\n")
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
