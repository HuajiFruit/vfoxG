package parser

import (
	"strings"

	"vfoxG/internal/model"
)

func AvailablePlugins(out string) []model.PluginInfo {
	lines := strings.Split(out, "\n")
	plugins := make([]model.PluginInfo, 0)
	for _, line := range lines {
		plugin, ok := parseAvailablePluginLine(line)
		if ok {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

func parseAvailablePluginLine(line string) (model.PluginInfo, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "AVAILABLE PLUGINS") || strings.HasPrefix(line, "Use 'vfox") {
		return model.PluginInfo{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return model.PluginInfo{}, false
	}
	return model.PluginInfo{
		Name:       parts[0],
		IsOfficial: isOfficialPluginStatus(parts[1]),
		URL:        parts[2],
	}, true
}

func isOfficialPluginStatus(status string) bool {
	status = strings.TrimSpace(status)
	return status == "✓" || status == "√" || strings.EqualFold(status, "true") || strings.EqualFold(status, "yes")
}
