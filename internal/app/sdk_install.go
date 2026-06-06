package app

import (
	"fmt"
	"strings"
)

func (a *App) installVersion(name, version string) error {
	name, version, err := validateSdkNameAndVersion(name, version)
	if err != nil {
		return err
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

func validateSdkName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	return name, nil
}
