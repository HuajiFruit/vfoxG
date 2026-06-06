package app

import (
	"fmt"
	"strings"
)

func (a *App) addMissingPluginForVersionSearch(name string, searchErr error) (bool, error) {
	if searchErr == nil || !isPluginNotInstalledSearchError(searchErr.Error()) {
		return false, nil
	}
	if _, addErr := a.runVfoxCommandWithLock("add", name); addErr != nil {
		return false, fmt.Errorf("plugin %s is not installed and auto-add failed: %w", name, addErr)
	}
	return true, nil
}

func isPluginNotInstalledSearchError(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "plugin") &&
		strings.Contains(lower, "not installed")
}
