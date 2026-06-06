package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func formatSdkEnvironmentExport(snapshot sdkEnvironmentExport) string {
	var b strings.Builder
	writeLine := newExportLineWriter(&b)

	writeExportHeader(writeLine, snapshot)
	writePlatformSection(writeLine, snapshot)
	writeWarningsSection(writeLine, snapshot.Warnings)
	writeVfoxSdks(writeLine, snapshot.VfoxSdks)
	writeCustomSdks(writeLine, snapshot.CustomSdks)
	writeSystemSdks(writeLine, snapshot.SystemSdks)
	return b.String()
}

func newExportLineWriter(b *strings.Builder) func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		if len(args) == 0 {
			b.WriteString(format)
		} else {
			b.WriteString(fmt.Sprintf(format, args...))
		}
		b.WriteString("\r\n")
	}
}

func writeExportHeader(writeLine func(string, ...interface{}), snapshot sdkEnvironmentExport) {
	writeLine("vfoxG SDK Environment Export")
	writeLine("Generated: %s", snapshot.GeneratedAt.Format(time.RFC3339))
	writeLine("")
}

func writePlatformSection(writeLine func(string, ...interface{}), snapshot sdkEnvironmentExport) {
	writeLine("Platform")
	writeLine("  OS: %s", emptyFallback(snapshot.Platform.Name, snapshot.Platform.OS))
	writeLine("  Core: %s/%s", snapshot.Platform.CoreOS, snapshot.Platform.CoreArch)
	writeLine("  VFOX_HOME: %s", emptyFallback(snapshot.Platform.DownloadPath, "(empty)"))
	writeLine("  Default VFOX_HOME: %s", emptyFallback(snapshot.Platform.DefaultDownloadPath, "(empty)"))
	writeLine("  vfox in PATH: %s", yesNo(snapshot.VfoxInPath))
	writeLine("  SDK PATH override active: %s", yesNo(snapshot.PathOverride))
	writeLine("")
}

func writeWarningsSection(writeLine func(string, ...interface{}), warnings []string) {
	if len(warnings) == 0 {
		return
	}
	writeLine("Warnings")
	for _, warning := range warnings {
		writeLine("  - %s", warning)
	}
	writeLine("")
}

func writeVfoxSdks(writeLine func(string, ...interface{}), vfoxSdks []sdkEnvironmentVfoxSdk) {
	writeLine("Vfox SDKs")
	writeLine("Name | Version | Current | Path")
	writeLine("--- | --- | --- | ---")
	if len(vfoxSdks) == 0 {
		writeLine("(none) |  |  | ")
	} else {
		for _, sdk := range vfoxSdks {
			writeVfoxSdkRows(writeLine, sdk)
		}
	}
	writeLine("")
}

func writeVfoxSdkRows(writeLine func(string, ...interface{}), sdk sdkEnvironmentVfoxSdk) {
	details := vfoxExportVersionDetails(sdk)
	if len(details) == 0 {
		writeLine("%s | (none) | no | ", tableCell(sdk.Name))
	}
	for _, version := range details {
		current := "no"
		if version.IsCurrent {
			current = "yes"
		}
		path := strings.TrimSpace(sdk.VersionPaths[version.Version])
		if path == "" && version.IsCurrent && sdk.ActiveCustomPath != "" {
			path = sdk.ActiveCustomPath
		}
		writeLine("%s | %s | %s | %s", tableCell(sdk.Name), tableCell(version.Version), current, tableCell(path))
	}
}

func vfoxExportVersionDetails(sdk sdkEnvironmentVfoxSdk) []SdkVersionDetail {
	details := sdk.Detail.Versions
	if len(details) > 0 {
		return details
	}
	for _, version := range sdk.Versions {
		details = append(details, SdkVersionDetail{
			Version:   version.Version,
			IsCurrent: version.Version != "" && version.Version == sdk.Detail.Current,
		})
	}
	return details
}

func writeSystemSdks(writeLine func(string, ...interface{}), systemSdks []SdkInfo) {
	writeLine("System SDKs")
	writeLine("Name | Version | Executable")
	writeLine("--- | --- | ---")
	if len(systemSdks) == 0 {
		writeLine("(none) |  | ")
	} else {
		for _, sdk := range systemSdks {
			writeSdkInfoRows(writeLine, sdk.Name, sdk.Path, sdk.Versions)
		}
	}
}

func writeCustomSdks(writeLine func(string, ...interface{}), customSdks map[string][]SdkInfo) {
	writeLine("Custom SDKs")
	writeLine("Name | Version | Path")
	writeLine("--- | --- | ---")
	if len(customSdks) == 0 {
		writeLine("(none) |  | ")
		writeLine("")
		return
	}
	names := make([]string, 0, len(customSdks))
	for name := range customSdks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		list := append([]SdkInfo(nil), customSdks[name]...)
		sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
		if len(list) == 0 {
			writeLine("%s | (none) | ", tableCell(name))
			continue
		}
		for _, sdk := range list {
			writeSdkInfoRows(writeLine, name, sdk.Path, sdk.Versions)
		}
	}
	writeLine("")
}

func writeSdkInfoRows(writeLine func(string, ...interface{}), name string, path string, versions []SdkVersion) {
	if len(versions) == 0 {
		writeLine("%s | (unknown) | %s", tableCell(name), tableCell(path))
		return
	}
	for _, version := range versions {
		writeLine("%s | %s | %s", tableCell(name), tableCell(emptyFallback(version.Version, "(unknown)")), tableCell(path))
	}
}

func tableCell(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "/")
	return strings.TrimSpace(value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
