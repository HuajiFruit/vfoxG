package app

import (
	"fmt"
	"os"
	"strings"
)

func (a *App) useVersion(name, version string) (string, error) {
	name, version, err := validateSdkNameAndVersion(name, version)
	if err != nil {
		return "", err
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()

	return a.useVersionUnlocked(name, version)
}

func (a *App) useVersionUnlocked(name, version string) (string, error) {
	if activeCustomPath, err := a.getActiveCustomSdk(name); err == nil && activeCustomPath != "" {
		a.removeJunctionIfExists(a.getVfoxHomePath("sdks", name))
	}

	// Use the unlocked command runner so the outer use flow owns the task lock.
	if _, err := a.runVfoxCommand("use", "--global", name+"@"+version); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	runtimeRoot, err := a.resolveVersionRuntimeRootUnlocked(name, version)
	if err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, runtimeRoot); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	if err := a.refreshActiveSdkPathOverride(name); err != nil {
		a.emitEvent("vfox-log", "[APP WARN] Failed to refresh SDK PATH override: "+err.Error())
	}

	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")
	return "ok", nil
}

func (a *App) unuseVersion(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()

	return a.unuseVersionUnlocked(name)
}

func (a *App) unuseVersionUnlocked(name string) (string, error) {
	if err := a.clearGlobalSdkSelectionUnlocked(name); err != nil {
		a.emitEvent("vfox-log", "[EXIT ERROR] "+err.Error())
		return "", err
	}
	a.removeJunctionIfExists(a.getVfoxHomePath("sdks", name))
	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")
	return "ok", nil
}

func (a *App) clearGlobalSdkSelection(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	return a.clearGlobalSdkSelectionUnlocked(name)
}

func (a *App) clearGlobalSdkSelectionUnlocked(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	var cmdErr error
	if _, err := a.runVfoxCommand("unuse", "--global", name); err != nil {
		cmdErr = err
	}
	if err := a.removeGlobalSdkSelectionFromConfig(name); err != nil {
		return err
	}
	if cmdErr != nil && !isBenignUnuseError(cmdErr) {
		return cmdErr
	}
	return nil
}

func isBenignUnuseError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no current") ||
		strings.Contains(message, "not installed") ||
		strings.Contains(message, "not supported")
}

func (a *App) removeGlobalSdkSelectionFromConfig(name string) error {
	configPath := a.getVfoxHomePath(".vfox.toml")
	if strings.TrimSpace(configPath) == "" {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	updated, changed := removeSdkSelectionFromVfoxToml(string(data), name)
	if !changed {
		return nil
	}
	return os.WriteFile(configPath, []byte(updated), 0644)
}

func removeSdkSelectionFromVfoxToml(data string, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return data, false
	}
	lines := strings.SplitAfter(data, "\n")
	var b strings.Builder
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if isSdkSelectionConfigLine(trimmed, name) {
			changed = true
			continue
		}
		b.WriteString(line)
	}
	return b.String(), changed
}

func isSdkSelectionConfigLine(line string, name string) bool {
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}
	return strings.HasPrefix(line, name+" ") || strings.HasPrefix(line, name+"=")
}
