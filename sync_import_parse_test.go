package main

import (
	"strings"
	"testing"
)

func TestParseSdkEnvironmentImport(t *testing.T) {
	report := strings.Join([]string{
		"vfoxG SDK Environment Export",
		"",
		"Vfox SDKs",
		"Name | Version | Current | Path",
		"--- | --- | --- | ---",
		"python | 3.14.4 | yes | C:\\vfox-home\\cache\\python",
		"",
		"Custom SDKs",
		"Name | Version | Path",
		"--- | --- | ---",
		"golang | 1.26.3 | D:\\go\\bin\\go.exe",
		"nodejs | 24.15.0 | D:\\node\\node.exe",
		"",
		"System SDKs",
		"Name | Version | Executable",
		"--- | --- | ---",
		"git | 2.54.0 | D:\\Git\\cmd\\git.exe",
	}, "\n")

	rows, warnings := parseSdkEnvironmentImport(report)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d import rows, want 3: %+v", len(rows), rows)
	}
	if rows[0].Kind != "vfox" || rows[0].Name != "python" || rows[0].Version != "3.14.4" || !rows[0].Current {
		t.Fatalf("unexpected vfox row: %+v", rows[0])
	}
	if rows[1].Kind != "custom" || rows[1].Name != "golang" || rows[1].Version != "1.26.3" || rows[1].Path != "D:\\go\\bin\\go.exe" {
		t.Fatalf("unexpected custom row: %+v", rows[1])
	}
	if rows[2].Kind != "custom" || rows[2].Name != "nodejs" || rows[2].Path != "D:\\node\\node.exe" {
		t.Fatalf("unexpected custom row: %+v", rows[2])
	}
}

func TestIsUnknownSdkVersion(t *testing.T) {
	for _, version := range []string{"", "unknown", "UNKNOWN", "(unknown)", " (unknown) "} {
		if !isUnknownSdkVersion(version) {
			t.Fatalf("isUnknownSdkVersion(%q) = false, want true", version)
		}
	}
	if isUnknownSdkVersion("1.26.3") {
		t.Fatal("isUnknownSdkVersion(\"1.26.3\") = true, want false")
	}
}
