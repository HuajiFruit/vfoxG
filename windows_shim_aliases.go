//go:build windows

package main

import "strings"

func windowsSafeShimName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if name == "" {
		return "sdk"
	}
	return name
}

func windowsUniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func windowsSDKShimAliases(pluginName string) []string {
	aliases := []string{pluginName}
	for _, def := range systemSDKDefs {
		if def.Name == pluginName {
			aliases = append(aliases, def.Exe)
			break
		}
	}

	switch strings.ToLower(pluginName) {
	case "python":
		aliases = append(aliases, "python3", "pip", "pip3")
	case "nodejs":
		aliases = append(aliases, "npm", "npx", "corepack")
	case "java":
		aliases = append(aliases, "javac", "jar", "jshell", "jlink", "jpackage", "keytool")
	case "golang":
		aliases = append(aliases, "gofmt")
	case "rust":
		aliases = append(aliases, "cargo", "rustdoc", "rustup")
	case "ruby":
		aliases = append(aliases, "gem", "bundle", "bundler", "irb", "rake")
	case "php":
		aliases = append(aliases, "composer")
	case "perl":
		aliases = append(aliases, "cpan")
	}
	return windowsUniqueStrings(aliases)
}
