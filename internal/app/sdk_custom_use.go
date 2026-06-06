package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) useCustomSdk(name string, exePath string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	if err := validateSDKExecutablePath(exePath); err != nil {
		return "", err
	}
	if err := a.ensureVfoxHomeDir(); err != nil {
		return "", err
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return "", err
	}
	defer releaseTask()

	return a.useCustomSdkUnlocked(name, exePath)
}

func (a *App) useCustomSdkUnlocked(name string, exePath string) (string, error) {
	root := a.getSdkRoot(exePath)

	// Go reports Windows junctions as ModeIrregular, so vfox can fail to resolve
	// them as runtime packages. Clearing vfox first prevents it from deleting the
	// custom link when we replace the managed SDK entrypoint.
	if err := a.clearGlobalSdkSelectionUnlocked(name); err != nil {
		return "", err
	}

	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	a.removeJunctionIfExists(sdkLinkPath)
	if err := a.ensureJunction(sdkLinkPath, root); err != nil {
		return "", fmt.Errorf("failed to create SDK junction: %v", err)
	}

	a.emitEvent("vfox-log", fmt.Sprintf("Activating %s (system)...", name))
	a.emitEvent("vfox-log", "[DONE]")
	a.emitEvent("sdk-list-changed")

	return "ok", nil
}

func (a *App) getActiveCustomSdk(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("plugin name cannot be empty")
	}
	sdkLinkPath := a.getVfoxHomePath("sdks", name)
	if sdkLinkPath == "" {
		return "", fmt.Errorf("unable to resolve vfox home directory")
	}

	fi, err := os.Lstat(sdkLinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0 {
		target, err := os.Readlink(sdkLinkPath)
		if err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(sdkLinkPath), target)
			}
			// Targets under any managed vfox root are not user-provided SDKs.
			if a.isVfoxManagedPath(target) {
				return "", nil
			}
			// Try to find matching SDK path
			customSdks := a.getNonVfoxSdksMap()
			cleanTarget := filepath.Clean(target)
			for _, sdk := range customSdks[name] {
				if strings.EqualFold(filepath.Clean(a.getSdkRoot(sdk.Path)), cleanTarget) {
					return sdk.Path, nil
				}
			}
			return target, nil
		}
	}

	return "", nil
}
