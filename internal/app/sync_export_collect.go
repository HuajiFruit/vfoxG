package app

import (
	"fmt"
	"sort"
	"time"
)

func (a *App) collectSdkEnvironmentExport(generatedAt time.Time) sdkEnvironmentExport {
	snapshot := sdkEnvironmentExport{
		GeneratedAt: generatedAt,
		Platform:    a.getPlatformInfo(),
		CustomSdks:  a.getNonVfoxSdksMap(),
	}

	if inPath, err := a.checkVfoxInPath(); err == nil {
		snapshot.VfoxInPath = inPath
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Unable to check vfox PATH status: "+err.Error())
	}
	snapshot.PathOverride = a.checkAnyPathOverride()

	if vfoxSdks, err := a.getInstalledSdks(); err == nil {
		sort.Slice(vfoxSdks, func(i, j int) bool { return vfoxSdks[i].Name < vfoxSdks[j].Name })
		for _, sdk := range vfoxSdks {
			exportSdk := sdkEnvironmentVfoxSdk{
				Name:         sdk.Name,
				Versions:     sdk.Versions,
				VersionPaths: make(map[string]string),
				Detail:       SdkDetail{Name: sdk.Name},
			}
			if detail, err := a.getSdkDetail(sdk.Name); err == nil {
				exportSdk.Detail = detail
				for _, version := range detail.Versions {
					if path, err := a.getVersionPath(sdk.Name, version.Version); err == nil {
						exportSdk.VersionPaths[version.Version] = path
					}
				}
			} else {
				snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("Unable to load vfox SDK detail for %s: %v", sdk.Name, err))
			}
			if activeCustomPath, err := a.getActiveCustomSdk(sdk.Name); err == nil {
				exportSdk.ActiveCustomPath = activeCustomPath
			}
			snapshot.VfoxSdks = append(snapshot.VfoxSdks, exportSdk)
		}
	} else {
		snapshot.Warnings = append(snapshot.Warnings, "Unable to load vfox SDK list: "+err.Error())
	}

	a.scanSystemSdks()
	snapshot.SystemSdks = a.getCachedSystemSdks()
	sort.Slice(snapshot.SystemSdks, func(i, j int) bool { return snapshot.SystemSdks[i].Name < snapshot.SystemSdks[j].Name })

	return snapshot
}
