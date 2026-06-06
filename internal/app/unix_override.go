//go:build !windows

package app

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const vfoxSDKPathMarkerPrefix = "vfoxG SDK PATH "

func (a *App) hijackSystemPath(name string, exePath string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return err
	}
	if strings.TrimSpace(exePath) != "" {
		if err := validateSDKExecutablePath(exePath); err != nil {
			return err
		}
		sdkRoot := a.getSdkRoot(exePath)
		sdkLinkPath := a.getVfoxHomePath("sdks", name)
		if err := a.ensureJunction(sdkLinkPath, sdkRoot); err != nil {
			return err
		}
	}

	sdkPath := a.getVfoxHomePath("sdks", name)
	return unixWritePathBlock(unixSDKMarkerLabel(name), unixSDKPathEntries(sdkPath))
}

func (a *App) restoreSystemPath(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	return unixRemoveManagedBlock(unixSDKMarkerLabel(name))
}

func (a *App) detachPluginPathOverrideFiles(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if sdkLinkPath == "" {
		return fmt.Errorf("unable to resolve vfox home directory")
	}
	a.removeJunctionIfExists(sdkLinkPath)
	return nil
}

func (a *App) hijackPluginSystemPath(pluginName string) error {
	m := a.getNonVfoxSdksMap()
	if list, ok := m[pluginName]; ok && len(list) > 0 {
		return a.hijackSystemPath(pluginName, list[0].Path)
	}
	return a.hijackSystemPath(pluginName, "")
}

func (a *App) restorePluginSystemPath(pluginName string) error {
	return a.restoreSystemPath(pluginName)
}

func (a *App) refreshActiveSdkPathOverride(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || !a.checkPluginPathOverride(name) {
		return nil
	}
	sdkPath := a.getVfoxHomePath("sdks", name)
	if strings.TrimSpace(sdkPath) == "" {
		return fmt.Errorf("%s: unable to resolve SDK PATH entry", name)
	}
	return unixWritePathBlock(unixSDKMarkerLabel(name), unixSDKPathEntries(sdkPath))
}

func (a *App) refreshPathOverridesAfterVfoxHomeChange(oldHome string) error {
	names, err := unixManagedSDKPathOverrideNames()
	if err != nil {
		return err
	}

	var errs []error
	for _, name := range names {
		sdkPath := a.getVfoxHomePath("sdks", name)
		if strings.TrimSpace(sdkPath) == "" {
			errs = append(errs, fmt.Errorf("%s: unable to resolve SDK PATH entry", name))
			continue
		}
		if err := unixWritePathBlock(unixSDKMarkerLabel(name), unixSDKPathEntries(sdkPath)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func unixManagedSDKPathOverrideNames() ([]string, error) {
	startPrefix := "# >>> " + vfoxSDKPathMarkerPrefix
	seen := make(map[string]bool)
	var names []string
	var lastErr error
	for _, profilePath := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimRight(line, "\r"))
			if !strings.HasPrefix(line, startPrefix) || !strings.HasSuffix(line, ">>>") {
				continue
			}
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, startPrefix), ">>>"))
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	if len(names) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return names, nil
}

func (a *App) checkPluginWin11CompatMode(pluginName string) bool {
	return unixManagedBlockExists(unixSDKMarkerLabel(pluginName))
}

func (a *App) checkWin11CompatMode() bool {
	for _, profile := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profile)
		if err == nil && strings.Contains(string(data), "# >>> "+vfoxSDKPathMarkerPrefix) {
			return true
		}
	}
	return false
}

func unixSDKMarkerLabel(pluginName string) string {
	return vfoxSDKPathMarkerPrefix + strings.TrimSpace(pluginName)
}
