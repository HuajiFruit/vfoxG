package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestPluginNameRegex(t *testing.T) {
	re := regexp.MustCompile(`^[├└]─┬(.+)`)

	tests := []struct {
		line     string
		expected string
	}{
		{"├─┬golang", "golang"},
		{"└─┬java", "java"},
	}

	for _, tt := range tests {
		match := re.FindStringSubmatch(tt.line)
		if len(match) < 2 || match[1] != tt.expected {
			t.Errorf("line %q: got %v, want %q", tt.line, match, tt.expected)
		}
	}
}

func TestVersionRegex(t *testing.T) {
	re := regexp.MustCompile(`^[│ ]\s*[├└]──(.+)`)

	tests := []struct {
		line     string
		expected string
	}{
		{"│  └──1.26.3", "1.26.3"},
		{"  ├──25.0.2+10", "25.0.2+10"},
		{"  └──21.0.2+13", "21.0.2+13"},
	}

	for _, tt := range tests {
		match := re.FindStringSubmatch(tt.line)
		if len(match) < 2 || match[1] != tt.expected {
			t.Errorf("line %q: got %v, want %q", tt.line, match, tt.expected)
		}
	}
}

func TestParseLsOutput(t *testing.T) {
	// 模拟 vfox 1.0.11 ls 输出（ANSI 已剥离）
	out := `All installed sdk versions
├─┬golang
│ └──1.26.3
└─┬java
  ├──25.0.2+10
  └──21.0.2+13`

	pluginRe := regexp.MustCompile(`^[├└]─┬(.+)`)
	versionRe := regexp.MustCompile(`^[│ ]\s*[├└]──(.+)`)

	lines := strings.Split(out, "\n")
	var sdks []SdkInfo
	var currentSdk *SdkInfo

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if match := pluginRe.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				sdks = append(sdks, *currentSdk)
			}
			currentSdk = &SdkInfo{Name: match[1], Versions: []SdkVersion{}}
		} else if match := versionRe.FindStringSubmatch(line); len(match) > 1 {
			if currentSdk != nil {
				currentSdk.Versions = append(currentSdk.Versions, SdkVersion{Version: match[1]})
			}
		}
	}
	if currentSdk != nil {
		sdks = append(sdks, *currentSdk)
	}

	if len(sdks) != 2 {
		t.Fatalf("expected 2 sdks, got %d", len(sdks))
	}
	if sdks[0].Name != "golang" || len(sdks[0].Versions) != 1 || sdks[0].Versions[0].Version != "1.26.3" {
		t.Errorf("golang: %+v", sdks[0])
	}
	if sdks[1].Name != "java" || len(sdks[1].Versions) != 2 {
		t.Errorf("java: %+v", sdks[1])
	}
}

func TestParseLsSdkOutput(t *testing.T) {
	// 模拟 vfox 1.0.11 ls golang 输出
	out := "-> 1.26.3 <-- current"
	lines := strings.Split(out, "\n")

	var versions []SdkVersionDetail
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		isCurrent := strings.HasPrefix(line, "->")
		ver := strings.TrimPrefix(line, "-> ")
		ver = strings.TrimSuffix(ver, " <-- current")
		ver = strings.TrimSpace(ver)
		if ver == "" {
			continue
		}
		versions = append(versions, SdkVersionDetail{Version: ver, IsCurrent: isCurrent})
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
	if !versions[0].IsCurrent {
		t.Error("expected IsCurrent=true")
	}
	if versions[0].Version != "1.26.3" {
		t.Errorf("expected '1.26.3', got %q", versions[0].Version)
	}
}
