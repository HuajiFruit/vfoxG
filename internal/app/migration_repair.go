package app

import (
	"errors"
	"fmt"
	"strings"
)

func (a *App) repairMigratedSdkEntrypoints(oldHome string) {
	if err := a.repairCurrentVfoxSdkLinks(); err != nil {
		a.emitEvent("vfox-log", "[APP WARN] Failed to refresh migrated SDK links: "+err.Error())
	}
	if err := a.refreshPathOverridesAfterVfoxHomeChange(oldHome); err != nil {
		a.emitEvent("vfox-log", "[APP WARN] Failed to refresh migrated SDK PATH entries: "+err.Error())
	}
}

func (a *App) repairCurrentVfoxSdkLinks() error {
	selections, err := a.readVfoxGlobalSelections()
	if err != nil {
		return err
	}
	if len(selections) == 0 {
		return nil
	}

	installedNames := make(map[string]bool)
	if installedSdks, err := a.getInstalledSdksUnlocked(); err == nil {
		for _, sdk := range installedSdks {
			installedNames[strings.ToLower(strings.TrimSpace(sdk.Name))] = true
		}
	}

	var errs []error
	for name, version := range selections {
		if len(installedNames) > 0 && !installedNames[strings.ToLower(name)] {
			continue
		}
		if activeCustomPath, err := a.getActiveCustomSdk(name); err == nil && activeCustomPath != "" {
			continue
		}

		runtimeRoot, err := a.resolveVersionRuntimeRootUnlocked(name, version)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s@%s: %w", name, version, err))
			continue
		}
		sdkLinkPath := a.getVfoxHomePath("sdks", name)
		if sdkLinkPath == "" {
			errs = append(errs, fmt.Errorf("%s@%s: unable to resolve sdk link path", name, version))
			continue
		}
		a.removeJunctionIfExists(sdkLinkPath)
		if err := a.ensureJunction(sdkLinkPath, runtimeRoot); err != nil {
			errs = append(errs, fmt.Errorf("%s@%s: %w", name, version, err))
		}
	}
	return errors.Join(errs...)
}
