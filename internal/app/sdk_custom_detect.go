package app

import (
	"fmt"
	"os"
	"strings"
)

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

func (a *App) detectSdkPathVersion(name string, exePath string) string {
	name = strings.TrimSpace(name)
	exePath = strings.TrimSpace(exePath)
	if name == "" || exePath == "" {
		return "unknown"
	}
	for _, def := range systemSDKDefs {
		if def.Name == name {
			version := a.tryGetVersion(exePath, def.VerArgs)
			if version != "" {
				return version
			}
			break
		}
	}
	return "unknown"
}
