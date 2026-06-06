package app

import (
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
)

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
	versionPath, err := a.getVersionPath(name, version)
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
