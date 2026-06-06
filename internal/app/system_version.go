package app

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	clean := cleanVersionOutputLine(raw)
	exe = normalizeExecutableName(exe)

	switch exe {
	case "python":
		return strings.TrimPrefix(clean, "Python ")
	case "node":
		return strings.TrimPrefix(clean, "v")
	case "java":
		// java -version 输出到 stderr，格式: openjdk version "21.0.2"
		if quoted := quotedVersionPart(clean); quoted != "" {
			return quoted
		}
		return clean
	case "go":
		if part := versionField(clean, 2); part != "" {
			return strings.TrimPrefix(part, "go")
		}
		return clean
	case "rustc", "ruby", "php", "lua":
		if part := versionField(clean, 1); part != "" {
			return part
		}
		return clean
	case "dotnet", "zig":
		return clean
	case "perl":
		if part := versionAfterWord(clean, "version"); part != "" {
			return part
		}
		return clean
	case "git":
		return strings.TrimPrefix(clean, "git version ")
	case "docker":
		if part := versionField(clean, 2); part != "" {
			return strings.TrimSuffix(part, ",")
		}
		return clean
	case "gcc":
		if part := lastVersionField(clean); part != "" {
			return part
		}
		return clean
	case "cmake":
		return strings.TrimPrefix(clean, "cmake version ")
	default:
		return clean
	}
}

func cleanVersionOutputLine(raw string) string {
	clean := ansiRegex.ReplaceAllString(raw, "")
	clean, _, _ = strings.Cut(clean, "\n")
	clean = strings.TrimRight(clean, "\r")
	return strings.TrimSpace(clean)
}

func quotedVersionPart(clean string) string {
	if idx := strings.Index(clean, `"`); idx >= 0 {
		rest := clean[idx+1:]
		if idx2 := strings.Index(rest, `"`); idx2 >= 0 {
			return rest[:idx2]
		}
	}
	return ""
}

func versionField(clean string, index int) string {
	parts := strings.Fields(clean)
	if index >= 0 && index < len(parts) {
		return parts[index]
	}
	return ""
}

func lastVersionField(clean string) string {
	parts := strings.Fields(clean)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func versionAfterWord(clean string, marker string) string {
	parts := strings.Fields(clean)
	for i, part := range parts {
		if part == marker && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

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
