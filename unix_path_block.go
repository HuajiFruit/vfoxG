//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func unixSDKPathEntries(sdkPath string) []string {
	return []string{
		sdkPath,
		filepath.Join(sdkPath, "bin"),
		filepath.Join(sdkPath, "sbin"),
	}
}

func unixWritePathBlock(label string, paths []string) error {
	profilePath, err := unixShellProfilePath()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no PATH entries to write")
	}

	block := unixManagedPathBlock(label, paths)
	data, _ := os.ReadFile(profilePath)
	updated := unixRemoveManagedBlockFromString(string(data), label)
	updated = strings.TrimRight(updated, "\r\n")
	if updated != "" {
		updated += "\n\n"
	}
	updated += block
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(profilePath, []byte(updated), 0644)
}

func unixRemoveManagedBlock(label string) error {
	var lastErr error
	changed := false
	for _, profilePath := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
			}
			continue
		}
		updated := unixRemoveManagedBlockFromString(string(data), label)
		if updated == string(data) {
			continue
		}
		if err := os.WriteFile(profilePath, []byte(updated), 0644); err != nil {
			lastErr = err
			continue
		}
		changed = true
	}
	if lastErr != nil {
		return lastErr
	}
	if !changed {
		return nil
	}
	return nil
}

func unixManagedBlockExists(label string) bool {
	start, _ := unixManagedBlockMarkers(label)
	for _, profilePath := range unixShellProfileCandidates() {
		data, err := os.ReadFile(profilePath)
		if err == nil && strings.Contains(string(data), start) {
			return true
		}
	}
	return false
}

func unixManagedPathBlock(label string, paths []string) string {
	start, end := unixManagedBlockMarkers(label)
	var quoted []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		quoted = append(quoted, shellQuote(filepath.Clean(p)))
	}
	if len(quoted) == 0 {
		quoted = append(quoted, `"$PATH"`)
	} else {
		quoted = append(quoted, `"$PATH"`)
	}

	return strings.Join([]string{
		start,
		"# Added by vfoxG. Remove this block from vfoxG settings.",
		"export PATH=" + strings.Join(quoted, ":"),
		end,
		"",
	}, "\n")
}

func unixManagedBlockMarkers(label string) (string, string) {
	cleanLabel := strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(label))
	return "# >>> " + cleanLabel + " >>>", "# <<< " + cleanLabel + " <<<"
}

func unixRemoveManagedBlockFromString(data string, label string) string {
	start, end := unixManagedBlockMarkers(label)
	for {
		startIdx := strings.Index(data, start)
		if startIdx < 0 {
			return data
		}
		endIdx := strings.Index(data[startIdx:], end)
		if endIdx < 0 {
			return data
		}
		endIdx = startIdx + endIdx + len(end)
		for endIdx < len(data) && (data[endIdx] == '\n' || data[endIdx] == '\r') {
			endIdx++
		}
		data = strings.TrimRight(data[:startIdx], "\r\n") + "\n" + data[endIdx:]
		data = strings.TrimLeft(data, "\r\n")
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
