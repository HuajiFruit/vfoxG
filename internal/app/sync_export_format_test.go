package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatSdkEnvironmentExport(t *testing.T) {
	report := formatSdkEnvironmentExport(testSdkEnvironmentExport())

	assertReportContains(t, report, []string{
		"vfoxG SDK Environment Export",
		"Generated: 2026-05-26T12:30:00Z",
		"vfox in PATH: yes",
		"SDK PATH override active: yes",
		"Vfox SDKs",
		"Name | Version | Current | Path",
		"python | 3.14.4 | yes",
		"Custom SDKs",
		"golang | 1.26.3",
		"System SDKs",
		"nodejs | 24.15.0",
	})
}

func assertReportContains(t *testing.T, report string, expectedFragments []string) {
	t.Helper()

	for _, want := range expectedFragments {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func testSdkEnvironmentExport() sdkEnvironmentExport {
	return sdkEnvironmentExport{
		GeneratedAt:  time.Date(2026, 5, 26, 12, 30, 0, 0, time.UTC),
		Platform:     testExportPlatformInfo(),
		VfoxInPath:   true,
		PathOverride: true,
		VfoxSdks: []sdkEnvironmentVfoxSdk{
			testExportVfoxSdk(),
		},
		SystemSdks: []SdkInfo{
			testExportSystemSdk(),
		},
		CustomSdks: map[string][]SdkInfo{
			"golang": {testExportCustomSdk()},
		},
	}
}

func testExportPlatformInfo() PlatformInfo {
	return PlatformInfo{
		OS:                  "windows",
		Name:                "Windows",
		CoreOS:              "windows",
		CoreArch:            "x86_64",
		DownloadPath:        filepath.Join("C:", "Users", "tester", "AppData", "Roaming", "vfoxG", "vfox-home"),
		DefaultDownloadPath: filepath.Join("C:", "Users", "tester", "AppData", "Roaming", "vfoxG", "vfox-home"),
	}
}

func testExportVfoxSdk() sdkEnvironmentVfoxSdk {
	return sdkEnvironmentVfoxSdk{
		Name: "python",
		Detail: SdkDetail{
			Name:    "python",
			Current: "3.14.4",
			Versions: []SdkVersionDetail{
				{Version: "3.14.4", IsCurrent: true},
			},
		},
		VersionPaths: map[string]string{"3.14.4": filepath.Join("C:", "sdk", "python")},
	}
}

func testExportSystemSdk() SdkInfo {
	return SdkInfo{
		Name:     "nodejs",
		Source:   "system",
		Path:     filepath.Join("D:", "NodeJS", "node.exe"),
		Versions: []SdkVersion{{Version: "24.15.0"}},
	}
}

func testExportCustomSdk() SdkInfo {
	return SdkInfo{
		Name:     "golang",
		Source:   "system",
		Path:     filepath.Join("D:", "go", "bin", "go.exe"),
		Versions: []SdkVersion{{Version: "1.26.3"}},
	}
}
