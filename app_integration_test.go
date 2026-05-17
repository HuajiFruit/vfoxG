//go:build integration

package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestRealVfoxLs(t *testing.T) {
	app := NewApp()

	out, err := app.RunVfoxCommand("ls")
	if err != nil {
		t.Fatalf("vfox ls failed: %v", err)
	}
	t.Logf("Raw ls output:\n%s", out)

	// Show each line with byte representation
	for i, line := range strings.Split(out, "\n") {
		t.Logf("  line %d: %q", i, line)
	}

	sdks, err := app.GetInstalledSdks()
	if err != nil {
		t.Fatalf("GetInstalledSdks failed: %v", err)
	}
	t.Logf("Parsed %d SDKs:", len(sdks))
	for _, s := range sdks {
		t.Logf("  %s: %d versions", s.Name, len(s.Versions))
		for _, v := range s.Versions {
			t.Logf("    - %s", v.Version)
		}
	}
}

func TestRealVfoxAvailable(t *testing.T) {
	app := NewApp()

	out, err := app.RunVfoxCommand("available")
	if err != nil {
		t.Fatalf("vfox available failed: %v", err)
	}
	t.Logf("Raw available output:\n%s", out)

	plugins, err := app.GetAvailablePlugins()
	if err != nil {
		t.Fatalf("GetAvailablePlugins failed: %v", err)
	}
	t.Logf("Parsed %d plugins", len(plugins))

	// Show a few examples
	for i, p := range plugins {
		if i >= 5 {
			break
		}
		t.Logf("  %s installed=%v url=%s", p.Name, p.IsAdded, p.URL)
	}
}

func TestRealVfoxSdkDetail(t *testing.T) {
	app := NewApp()

	// Get all SDKs first
	sdks, err := app.GetInstalledSdks()
	if err != nil {
		t.Fatalf("GetInstalledSdks: %v", err)
	}
	if len(sdks) == 0 {
		t.Skip("No SDKs installed")
	}

	for _, sdk := range sdks {
		t.Run(sdk.Name, func(t *testing.T) {
			// Raw output check
			raw, _ := app.RunVfoxCommand("ls", sdk.Name)
			t.Logf("Raw 'vfox ls %s': %q", sdk.Name, raw)

			rawCur, _ := app.RunVfoxCommand("current", sdk.Name)
			t.Logf("Raw 'vfox current %s': %q", sdk.Name, rawCur)

			detail, err := app.GetSdkDetail(sdk.Name)
			if err != nil {
				t.Fatalf("GetSdkDetail failed: %v", err)
			}
			t.Logf("Name: %s", detail.Name)
			t.Logf("Current: %s", detail.Current)
			t.Logf("Versions: %d", len(detail.Versions))
			for _, v := range detail.Versions {
				t.Logf("  %s (current=%v)", v.Version, v.IsCurrent)
			}

			if detail.Current == "" && len(detail.Versions) > 0 {
				t.Error("No current version detected but versions exist")
			}
		})
	}
}

func TestVfoxCommandTimeout(t *testing.T) {
	app := NewApp()

	// vfox ls should complete quickly
	out, err := app.RunVfoxCommand("ls")
	if err != nil {
		t.Errorf("vfox ls should not time out: %v", err)
	}
	if out == "" {
		t.Error("vfox ls returned empty output")
	}
}

func TestRegexDirectly(t *testing.T) {
	// Test with REAL vfox output patterns
	app := NewApp()
	out, _ := app.RunVfoxCommand("ls")

	pluginRe := regexp.MustCompile(`^[├└]─┬(.+)`)
	versionRe := regexp.MustCompile(`^[│ ]\s*[├└]──(.*)`)

	matchedAny := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if pluginRe.MatchString(line) {
			matchedAny = true
			m := pluginRe.FindStringSubmatch(line)
			t.Logf("PLUGIN match: %q → name=%q", line, m[1])
		}
		if versionRe.MatchString(line) {
			matchedAny = true
			m := versionRe.FindStringSubmatch(line)
			t.Logf("VERSION match: %q → ver=%q", line, m[1])
		}
	}
	if !matchedAny {
		t.Error("Neither regex matched any line in vfox ls output!")
		fmt.Println("Full output:", out)
	}
}
