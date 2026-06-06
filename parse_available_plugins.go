package main

import "strings"

func parseAvailablePluginsOutput(out string) []PluginInfo {
	lines := strings.Split(out, "\n")
	plugins := make([]PluginInfo, 0)
	for _, line := range lines {
		plugin, ok := parseAvailablePluginLine(line)
		if ok {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

func parseAvailablePluginLine(line string) (PluginInfo, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "AVAILABLE PLUGINS") || strings.HasPrefix(line, "Use 'vfox") {
		return PluginInfo{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return PluginInfo{}, false
	}
	return PluginInfo{
		Name:       parts[0],
		IsOfficial: isOfficialPluginStatus(parts[1]),
		URL:        parts[2],
	}, true
}

func isOfficialPluginStatus(status string) bool {
	status = strings.TrimSpace(status)
	return status == "✓" || status == "√" || strings.EqualFold(status, "true") || strings.EqualFold(status, "yes")
}
