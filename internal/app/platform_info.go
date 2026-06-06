package app

import (
	"os"
	"path/filepath"
	stdruntime "runtime"
)

func (a *App) getPlatformInfo() PlatformInfo {
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
