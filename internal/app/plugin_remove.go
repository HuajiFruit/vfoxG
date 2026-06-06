package app

import (
	"fmt"
	"os"
	"strings"
)

type pluginRemovalPlan struct {
	Name                string
	CustomSdks          map[string][]SdkInfo
	KeptCustomSdks      []SdkInfo
	RestorePathOverride bool
	DetachPathOverride  bool
}

func (a *App) removePlugin(name string) error {
	return a.removePluginWithOptions(name, "")
}

func (a *App) removePluginWithOptions(name string, keepCustomSdkPath string) (err error) {
	releaseTask, lockErr := a.tryStartVfoxTask()
	if lockErr != nil {
		a.emitEvent("vfox-busy")
		return lockErr
	}
	defer releaseTask()

	removalPlan, err := a.planPluginRemoval(name, keepCustomSdkPath)
	if err != nil {
		return err
	}

	restoreDetachedOverride, err := a.preparePluginPathOverrideRemoval(removalPlan)
	if err != nil {
		return err
	}
	if restoreDetachedOverride {
		defer func() {
			if err != nil {
				err = a.restoreDetachedPluginPathOverride(removalPlan, err)
			}
		}()
		defer func() {
			if err == nil {
				restoreDetachedOverride = false
			}
		}()
	}

	a.uninstallPluginVersions(removalPlan.Name)
	a.clearPluginSelections(removalPlan.Name)

	if err := a.runVfoxWithProgress([]string{"remove", "-y", removalPlan.Name}); err != nil {
		return err
	}
	if err := a.savePluginRemovalCustomSdks(removalPlan); err != nil {
		return err
	}

	if len(removalPlan.KeptCustomSdks) > 0 {
		restoreDetachedOverride = false
		if err := a.hijackSystemPath(removalPlan.Name, removalPlan.KeptCustomSdks[0].Path); err != nil {
			return err
		}
		a.emitEvent("sdk-list-changed")
		return nil
	}

	a.removePluginVfoxLink(removalPlan.Name)
	return nil
}

func (a *App) planPluginRemoval(name string, keepCustomSdkPath string) (pluginRemovalPlan, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return pluginRemovalPlan{}, fmt.Errorf("plugin name cannot be empty")
	}
	keepCustomSdkPath = strings.TrimSpace(keepCustomSdkPath)

	customSdks := a.getNonVfoxSdksMap()
	keptCustomSdks, err := customSdksToKeep(customSdks[name], keepCustomSdkPath)
	if err != nil {
		return pluginRemovalPlan{}, err
	}
	if len(keptCustomSdks) > 0 {
		if err := validateSDKExecutablePath(keptCustomSdks[0].Path); err != nil {
			return pluginRemovalPlan{}, err
		}
	}

	return pluginRemovalPlan{
		Name:                name,
		CustomSdks:          customSdks,
		KeptCustomSdks:      keptCustomSdks,
		RestorePathOverride: len(keptCustomSdks) == 0 && a.checkPluginPathOverride(name),
		DetachPathOverride:  len(keptCustomSdks) > 0,
	}, nil
}

func (a *App) preparePluginPathOverrideRemoval(removalPlan pluginRemovalPlan) (bool, error) {
	if removalPlan.RestorePathOverride {
		if err := a.restoreSystemPath(removalPlan.Name); err != nil {
			return false, err
		}
	}
	if removalPlan.DetachPathOverride {
		if err := a.detachPluginPathOverrideFiles(removalPlan.Name); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (a *App) restoreDetachedPluginPathOverride(removalPlan pluginRemovalPlan, originalErr error) error {
	if len(removalPlan.KeptCustomSdks) == 0 {
		return originalErr
	}
	if restoreErr := a.hijackSystemPath(removalPlan.Name, removalPlan.KeptCustomSdks[0].Path); restoreErr != nil {
		return fmt.Errorf("%w; also failed to restore kept custom SDK path override: %v", originalErr, restoreErr)
	}
	return originalErr
}

func (a *App) uninstallPluginVersions(name string) {
	installedSdks, err := a.getInstalledSdksUnlocked()
	if err != nil {
		return
	}
	for _, sdk := range installedSdks {
		if sdk.Name == name && sdk.Source == "vfox" {
			for _, version := range sdk.Versions {
				_ = a.runVfoxWithProgress([]string{"uninstall", name + "@" + version.Version})
			}
			return
		}
	}
}

func (a *App) clearPluginSelections(name string) {
	_, _ = a.runVfoxCommand("unuse", "-g", name)
	_, _ = a.runVfoxCommand("unuse", "-p", name)
	_, _ = a.runVfoxCommand("unuse", "-s", name)
}

func (a *App) savePluginRemovalCustomSdks(removalPlan pluginRemovalPlan) error {
	if len(removalPlan.KeptCustomSdks) > 0 {
		removalPlan.CustomSdks[removalPlan.Name] = removalPlan.KeptCustomSdks
	} else if _, ok := removalPlan.CustomSdks[removalPlan.Name]; ok {
		delete(removalPlan.CustomSdks, removalPlan.Name)
	}
	return a.saveNonVfoxSdksMap(removalPlan.CustomSdks)
}

func (a *App) removePluginVfoxLink(name string) {
	if vfoxLinkPath := a.getVfoxHomePath("sdks", name); vfoxLinkPath != "" {
		_ = os.RemoveAll(vfoxLinkPath)
	}
}
