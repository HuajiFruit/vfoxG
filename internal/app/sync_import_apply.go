package app

import (
	"fmt"
	"os"
	"strings"

	"vfoxG/internal/parser"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type vfoxImportTarget struct {
	Name    string
	Version string
	Current bool
}

func (a *App) importSdkEnvironmentFromTxt() (SdkEnvironmentImportResult, error) {
	result := SdkEnvironmentImportResult{}
	if a.ctx == nil {
		return result, fmt.Errorf("application context is not ready")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import SDK environment",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return result, err
	}
	path = strings.TrimSpace(path)
	result.Path = path
	if path == "" {
		return result, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	rows, warnings := parseSdkEnvironmentImport(string(data))
	result.Warnings = append(result.Warnings, warnings...)

	customSdks := a.getNonVfoxSdksMap()
	vfoxTargets := collectVfoxImportTargets(rows, &result)
	a.importCustomSdkRows(rows, customSdks, &result)

	if result.ImportedCustomSdks > 0 {
		if err := a.saveNonVfoxSdksMap(customSdks); err != nil {
			return result, err
		}
		a.emitEvent("sdk-list-changed")
	}
	if len(vfoxTargets) > 0 {
		if err := a.installVfoxImportTargets(vfoxTargets, &result); err != nil {
			return result, err
		}
		a.emitEvent("sdk-list-changed")
	}
	return result, nil
}

func collectVfoxImportTargets(rows []sdkEnvironmentImportRow, result *SdkEnvironmentImportResult) []vfoxImportTarget {
	var vfoxTargets []vfoxImportTarget
	seenVfoxTargets := make(map[string]bool)
	vfoxTargetIndexes := make(map[string]int)

	for _, row := range rows {
		if row.Kind != "vfox" {
			continue
		}
		name := strings.TrimSpace(row.Name)
		version := parser.NormalizeSdkVersion(row.Version)
		if name == "" || isUnknownSdkVersion(version) {
			result.SkippedVfoxSdks++
			continue
		}

		result.VfoxSdksFound++
		key := strings.ToLower(name) + "@" + strings.ToLower(version)
		if !seenVfoxTargets[key] {
			seenVfoxTargets[key] = true
			vfoxTargetIndexes[key] = len(vfoxTargets)
			vfoxTargets = append(vfoxTargets, vfoxImportTarget{Name: name, Version: version, Current: row.Current})
			continue
		}
		if row.Current {
			vfoxTargets[vfoxTargetIndexes[key]].Current = true
		}
	}
	return vfoxTargets
}

func (a *App) importCustomSdkRows(rows []sdkEnvironmentImportRow, customSdks map[string][]SdkInfo, result *SdkEnvironmentImportResult) {
	for _, row := range rows {
		if row.Kind != "custom" {
			continue
		}
		a.importCustomSdkRow(row, customSdks, result)
	}
}

func (a *App) importCustomSdkRow(row sdkEnvironmentImportRow, customSdks map[string][]SdkInfo, result *SdkEnvironmentImportResult) {
	if row.Name == "" || row.Path == "" {
		result.SkippedCustomSdks++
		result.Warnings = append(result.Warnings, "Skipped custom SDK row with missing name or path.")
		return
	}
	if err := validateSDKExecutablePath(row.Path); err != nil {
		result.SkippedCustomSdks++
		result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s at %s: %v", row.Name, row.Path, err))
		return
	}
	for _, existing := range customSdks[row.Name] {
		if samePath(existing.Path, row.Path) {
			result.SkippedCustomSdks++
			return
		}
	}

	version := strings.TrimSpace(row.Version)
	if isUnknownSdkVersion(version) {
		version = a.detectSdkPathVersion(row.Name, row.Path)
	}
	if version == "" {
		version = "unknown"
	}
	customSdks[row.Name] = append(customSdks[row.Name], SdkInfo{
		Name:     row.Name,
		Source:   "system",
		Path:     row.Path,
		Versions: []SdkVersion{{Version: version}},
	})
	result.ImportedCustomSdks++
}

func (a *App) installVfoxImportTargets(vfoxTargets []vfoxImportTarget, result *SdkEnvironmentImportResult) error {
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	addedPlugins := a.loadAddedPluginSet(result)
	for _, target := range vfoxTargets {
		a.installVfoxImportTarget(target, addedPlugins, result)
	}
	return nil
}

func (a *App) loadAddedPluginSet(result *SdkEnvironmentImportResult) map[string]bool {
	addedPlugins := make(map[string]bool)
	plugins, err := a.getAddedPlugins()
	if err != nil {
		result.Warnings = append(result.Warnings, "Unable to list added vfox plugins: "+err.Error())
		return addedPlugins
	}
	for _, plugin := range plugins {
		addedPlugins[strings.ToLower(strings.TrimSpace(plugin))] = true
	}
	return addedPlugins
}

func (a *App) installVfoxImportTarget(target vfoxImportTarget, addedPlugins map[string]bool, result *SdkEnvironmentImportResult) {
	pluginKey := strings.ToLower(target.Name)
	if !addedPlugins[pluginKey] {
		if err := a.runVfoxWithProgress([]string{"add", target.Name}); err != nil {
			result.SkippedVfoxSdks++
			result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s@%s: failed to add vfox plugin: %v", target.Name, target.Version, err))
			return
		}
		addedPlugins[pluginKey] = true
	}
	if err := a.installVersionUnlocked(target.Name, target.Version); err != nil {
		result.SkippedVfoxSdks++
		result.Warnings = append(result.Warnings, fmt.Sprintf("Skipped %s@%s: failed to install vfox SDK: %v", target.Name, target.Version, err))
		return
	}
	result.InstalledVfoxSdks++
	if target.Current {
		if _, err := a.useVersionUnlocked(target.Name, target.Version); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Installed %s@%s but failed to activate it: %v", target.Name, target.Version, err))
		}
	}
}
