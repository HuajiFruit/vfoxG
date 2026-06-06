//go:build windows

package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type windowsPathOverrideEntry struct {
	Version      int      `json:"Version,omitempty"`
	ShimDir      string   `json:"ShimDir,omitempty"`
	Aliases      []string `json:"Aliases,omitempty"`
	UserPaths    []string `json:"UserPaths,omitempty"`
	MachinePaths []string `json:"MachinePaths,omitempty"`
}

func (a *App) writeWindowsPathOverrideEntry(name string, aliases []string) error {
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if strings.TrimSpace(hijackFile) == "" {
		return fmt.Errorf("unable to resolve PATH override metadata file")
	}
	entries, err := readWindowsPathOverrideEntries(hijackFile)
	if err != nil {
		return err
	}
	entry := entries[name]
	entry.Version = 2
	entry.ShimDir = a.windowsPathShimDir()
	entry.Aliases = windowsUniqueStrings(aliases)
	entries[name] = entry
	return a.writeJSONFile(hijackFile, entries)
}

func (a *App) checkPluginWin11CompatMode(pluginName string) bool {
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if hijackFile == "" {
		return false
	}
	if _, err := os.Stat(hijackFile); err == nil {
		data, err := os.ReadFile(hijackFile)
		if err == nil {
			var parsed map[string]interface{}
			if json.Unmarshal(data, &parsed) == nil {
				_, exists := parsed[pluginName]
				return exists
			}
		}
	}
	return false
}

func (a *App) checkWin11CompatMode() bool {
	hijackFile := a.getVfoxHomePath("hijacked_paths.json")
	if hijackFile == "" {
		return false
	}
	if _, err := os.Stat(hijackFile); err == nil {
		data, err := os.ReadFile(hijackFile)
		if err == nil {
			var parsed map[string]interface{}
			if json.Unmarshal(data, &parsed) == nil {
				return len(parsed) > 0
			}
		}
	}
	return false
}

func readWindowsPathOverrideAliases(hijackFile string, name string) ([]string, bool) {
	entries, err := readWindowsPathOverrideEntries(hijackFile)
	if err != nil {
		return nil, false
	}
	entry, ok := entries[name]
	if !ok {
		return nil, false
	}
	aliases := windowsUniqueStrings(entry.Aliases)
	return aliases, len(aliases) > 0
}

func readWindowsPathOverrideEntries(hijackFile string) (map[string]windowsPathOverrideEntry, error) {
	result := make(map[string]windowsPathOverrideEntry)
	data, err := os.ReadFile(hijackFile)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return result, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for name, entryData := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var entry struct {
			Version      int             `json:"Version,omitempty"`
			ShimDir      string          `json:"ShimDir,omitempty"`
			Aliases      json.RawMessage `json:"Aliases"`
			UserPaths    []string        `json:"UserPaths,omitempty"`
			MachinePaths []string        `json:"MachinePaths,omitempty"`
		}
		if err := json.Unmarshal(entryData, &entry); err != nil {
			continue
		}
		result[name] = windowsPathOverrideEntry{
			Version:      entry.Version,
			ShimDir:      entry.ShimDir,
			Aliases:      decodeWindowsPathOverrideAliases(entry.Aliases),
			UserPaths:    entry.UserPaths,
			MachinePaths: entry.MachinePaths,
		}
	}
	return result, nil
}

func decodeWindowsPathOverrideAliases(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var aliases []string
	if err := json.Unmarshal(raw, &aliases); err == nil {
		return windowsUniqueStrings(aliases)
	}

	var legacyAliases []struct {
		Value []string `json:"value"`
	}
	if err := json.Unmarshal(raw, &legacyAliases); err != nil {
		return nil
	}
	for _, item := range legacyAliases {
		aliases = append(aliases, item.Value...)
	}
	return windowsUniqueStrings(aliases)
}
