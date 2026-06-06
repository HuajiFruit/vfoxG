package app

import (
	"fmt"
	"strings"

	"vfoxG/internal/parser"
)

func (a *App) getVersionPath(name, version string) (string, error) {
	name, version, err := validateSdkNameAndVersion(name, version)
	if err != nil {
		return "", err
	}
	return a.getVersionPathUnlocked(name, version)
}

func (a *App) searchVersions(name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{}, fmt.Errorf("plugin name cannot be empty")
	}
	out, err := a.runVfoxCommandWithLock("search", name)
	if err != nil {
		shouldRetry, addErr := a.addMissingPluginForVersionSearch(name, err)
		if addErr != nil {
			return []string{}, addErr
		}
		if shouldRetry {
			out, err = a.runVfoxCommandWithLock("search", name)
		}
		if err != nil {
			return []string{}, err
		}
	}

	return parser.SearchVersions(out), nil
}
