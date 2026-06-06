package app

import "strings"

func isUnknownSdkVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version == "" || strings.EqualFold(version, "unknown") || strings.EqualFold(version, "(unknown)")
}

func parseSdkEnvironmentImport(data string) ([]sdkEnvironmentImportRow, []string) {
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	section := ""
	currentCustomName := ""
	var rows []sdkEnvironmentImportRow
	var warnings []string

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" {
			continue
		}

		if newSection, ok := sdkEnvironmentImportSection(line); ok {
			section = newSection
			currentCustomName = ""
			continue
		}

		if handled := parseSdkEnvironmentImportTableRow(line, section, &rows); handled {
			continue
		}

		if parseLegacyCustomImportLine(rawLine, line, section, &currentCustomName, &rows) {
			continue
		}

		if parseLegacyVfoxImportLine(rawLine, line, section, &currentCustomName, &rows) {
			continue
		}
	}

	if len(rows) == 0 {
		warnings = append(warnings, "No SDK rows were found in the selected text file.")
	}
	return rows, warnings
}

func sdkEnvironmentImportSection(line string) (string, bool) {
	switch strings.ToLower(line) {
	case "vfox sdks":
		return "vfox", true
	case "custom sdks", "custom sdk paths":
		return "custom", true
	case "system sdks":
		return "system", true
	default:
		return "", false
	}
}

func parseSdkEnvironmentImportTableRow(line string, section string, rows *[]sdkEnvironmentImportRow) bool {
	parts, ok := parseExportTableLine(line)
	if !ok {
		return false
	}
	if len(parts) == 0 || isExportTableHeader(parts) {
		return true
	}

	switch section {
	case "vfox":
		appendVfoxImportTableRow(parts, rows)
	case "custom":
		appendCustomImportTableRow(parts, rows)
	}
	return true
}

func appendVfoxImportTableRow(parts []string, rows *[]sdkEnvironmentImportRow) {
	if len(parts) < 2 || strings.EqualFold(parts[0], "(none)") {
		return
	}
	*rows = append(*rows, sdkEnvironmentImportRow{
		Kind:    "vfox",
		Name:    parts[0],
		Version: parts[1],
		Current: len(parts) >= 3 && strings.EqualFold(parts[2], "yes"),
		Path:    tablePart(parts, 3),
	})
}

func appendCustomImportTableRow(parts []string, rows *[]sdkEnvironmentImportRow) {
	if len(parts) < 3 || strings.EqualFold(parts[0], "(none)") {
		return
	}
	*rows = append(*rows, sdkEnvironmentImportRow{
		Kind:    "custom",
		Name:    parts[0],
		Version: parts[1],
		Path:    parts[2],
	})
}

func parseLegacyCustomImportLine(rawLine string, line string, section string, currentCustomName *string, rows *[]sdkEnvironmentImportRow) bool {
	if section != "custom" {
		return false
	}
	if strings.HasPrefix(rawLine, "  ") && !strings.HasPrefix(rawLine, "    ") {
		*currentCustomName = line
		return true
	}
	if strings.HasPrefix(line, "- Path:") && *currentCustomName != "" {
		*rows = append(*rows, sdkEnvironmentImportRow{
			Kind: "custom",
			Name: *currentCustomName,
			Path: strings.TrimSpace(strings.TrimPrefix(line, "- Path:")),
		})
		return true
	}
	if strings.HasPrefix(line, "Version:") && canAttachLegacyCustomVersion(*rows) {
		(*rows)[len(*rows)-1].Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		return true
	}
	return false
}

func canAttachLegacyCustomVersion(rows []sdkEnvironmentImportRow) bool {
	return len(rows) > 0 && rows[len(rows)-1].Kind == "custom" && rows[len(rows)-1].Version == ""
}

func parseLegacyVfoxImportLine(rawLine string, line string, section string, currentCustomName *string, rows *[]sdkEnvironmentImportRow) bool {
	if section != "vfox" {
		return false
	}
	if strings.HasPrefix(rawLine, "  ") && !strings.HasPrefix(rawLine, "    ") {
		*currentCustomName = line
		return true
	}
	if strings.HasPrefix(line, "- ") && *currentCustomName != "" {
		version := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		version = strings.TrimSuffix(version, " (current)")
		*rows = append(*rows, sdkEnvironmentImportRow{
			Kind:    "vfox",
			Name:    *currentCustomName,
			Version: strings.TrimSpace(version),
			Current: strings.Contains(line, "(current)"),
		})
		return true
	}
	return false
}

func parseExportTableLine(line string) ([]string, bool) {
	if !strings.Contains(line, "|") {
		return nil, false
	}
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

func isExportTableHeader(parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	first := strings.TrimSpace(parts[0])
	if strings.EqualFold(first, "Name") {
		return true
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Trim(part, "- ") != "" {
			return false
		}
	}
	return true
}

func tablePart(parts []string, index int) string {
	if index >= len(parts) {
		return ""
	}
	return parts[index]
}
