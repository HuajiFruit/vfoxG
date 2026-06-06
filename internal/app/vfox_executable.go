package app

import (
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
)

func (a *App) getCoreDir() string {
	suffix := filepath.Join(coreOSName(), coreArchName())
	candidates := vfoxCoreDirCandidates()

	for _, candidate := range candidates {
		coreDir := filepath.Join(candidate, suffix)
		exePath := filepath.Join(coreDir, getVfoxExeName())
		if info, err := os.Stat(exePath); err == nil && !info.IsDir() {
			return coreDir
		}
	}

	baseDir, _ := filepath.Abs("core")
	return filepath.Join(baseDir, suffix)
}

func vfoxCoreDirCandidates() []string {
	var candidates []string
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "core"),
			filepath.Join(exeDir, "..", "Resources", "core"),
			filepath.Join(exeDir, "..", "..", "core"),
		)
	}
	if stdruntime.GOOS == "linux" {
		candidates = append(candidates, "/usr/lib/vfoxg/core")
	}
	if abs, err := filepath.Abs("core"); err == nil {
		candidates = append(candidates, abs)
	}
	return candidates
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
