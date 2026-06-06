package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

func (a *App) getVfoxHome() string {
	a.homeMu.RLock()
	path := strings.TrimSpace(a.vfoxHome)
	a.homeMu.RUnlock()
	if path != "" {
		return path
	}
	if envVfoxHome := strings.TrimSpace(os.Getenv("VFOX_HOME")); envVfoxHome != "" {
		if normalized, err := normalizeDownloadPath(envVfoxHome); err == nil {
			return normalized
		}
		return envVfoxHome
	}
	return a.defaultVfoxHome()
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
