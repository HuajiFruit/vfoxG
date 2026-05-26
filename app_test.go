package main

import (
	"os"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"strings"
	"testing"
	"time"
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
	tests := []struct {
		name        string
		line        string
		wantVersion string
		wantCurrent bool
		wantOK      bool
	}{
		{name: "ascii current marker", line: "-> 1.26.3 <-- current", wantVersion: "1.26.3", wantCurrent: true, wantOK: true},
		{name: "unicode current marker", line: "-> 1.26.3 <— current", wantVersion: "1.26.3", wantCurrent: true, wantOK: true},
		{name: "leading arrow only", line: "-> 1.26.3", wantVersion: "1.26.3", wantCurrent: true, wantOK: true},
		{name: "plain version", line: "1.26.2", wantVersion: "1.26.2", wantCurrent: false, wantOK: true},
		{name: "custom sys hidden", line: "custom-sys-python", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCurrent, gotOK := parseSdkDetailVersionLine(tt.line)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if gotVersion != tt.wantVersion {
				t.Fatalf("version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotCurrent != tt.wantCurrent {
				t.Fatalf("current = %v, want %v", gotCurrent, tt.wantCurrent)
			}
		})
	}
}

func TestParseCurrentSdkVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
	}{
		{name: "plain arrow", out: "-> v3.14.4\n", want: "3.14.4"},
		{name: "sdk arrow", out: "python -> v3.13.12\n", want: "3.13.12"},
		{name: "sdk colon", out: "python: v3.12.10\n", want: "3.12.10"},
		{name: "sdk at", out: "python@v3.11.9\n", want: "3.11.9"},
		{name: "not installed", out: "python not supported, error: python not installed\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCurrentSdkVersion("python", tt.out); got != tt.want {
				t.Fatalf("parseCurrentSdkVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSdkDetailOutputUsesSingleCurrentVersion(t *testing.T) {
	out := `-> 3.14.4 <-- current
-> 3.13.12 <-- current`

	detail := parseSdkDetailOutput("python", "3.13.12", out)
	if detail.Current != "3.13.12" {
		t.Fatalf("current = %q, want 3.13.12", detail.Current)
	}
	if len(detail.Versions) != 2 {
		t.Fatalf("got %d versions, want 2", len(detail.Versions))
	}
	if detail.Versions[0].IsCurrent {
		t.Fatal("first version should not be current")
	}
	if !detail.Versions[1].IsCurrent {
		t.Fatal("second version should be current")
	}
}

func TestParseSdkDetailOutputDropsAmbiguousCurrentMarkers(t *testing.T) {
	out := `-> 3.14.4 <-- current
-> 3.13.12 <-- current`

	detail := parseSdkDetailOutput("python", "", out)
	if detail.Current != "" {
		t.Fatalf("current = %q, want empty", detail.Current)
	}
	for _, version := range detail.Versions {
		if version.IsCurrent {
			t.Fatalf("version %q should not be current", version.Version)
		}
	}
}

func TestParseSdkDetailOutputUsesSingleMarkerFallback(t *testing.T) {
	out := `3.14.4
-> 3.13.12 <-- current`

	detail := parseSdkDetailOutput("python", "", out)
	if detail.Current != "3.13.12" {
		t.Fatalf("current = %q, want 3.13.12", detail.Current)
	}
	if !detail.Versions[1].IsCurrent {
		t.Fatal("marked version should be current")
	}
}

func TestRemoveSdkSelectionFromVfoxToml(t *testing.T) {
	input := "# keep this\r\npython = \"3.13.12\"\r\nnodejs = \"22.0.0\"\r\npython_extra = \"keep\"\r\n"
	got, changed := removeSdkSelectionFromVfoxToml(input, "python")
	if !changed {
		t.Fatal("expected config to change")
	}
	want := "# keep this\r\nnodejs = \"22.0.0\"\r\npython_extra = \"keep\"\r\n"
	if got != want {
		t.Fatalf("updated config = %q, want %q", got, want)
	}
}

func TestRemoveSdkSelectionFromVfoxTomlNoMatch(t *testing.T) {
	input := "nodejs = \"22.0.0\"\n"
	got, changed := removeSdkSelectionFromVfoxToml(input, "python")
	if changed {
		t.Fatal("did not expect config to change")
	}
	if got != input {
		t.Fatalf("updated config = %q, want original", got)
	}
}

func TestSdkRootHasExecutableFindsNestedPython(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "python-3.14.5")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	exeName := "python"
	if stdruntime.GOOS == "windows" {
		exeName = "python.exe"
	}
	if err := os.WriteFile(filepath.Join(root, exeName), []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if !sdkRootHasExecutable(root, "python") {
		t.Fatal("expected python executable to be detected")
	}
}

func TestSdkRootHasExecutableFindsBinExecutable(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	exeName := "node"
	if stdruntime.GOOS == "windows" {
		exeName = "node.exe"
	}
	if err := os.WriteFile(filepath.Join(bin, exeName), []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if !sdkRootHasExecutable(tmp, "nodejs") {
		t.Fatal("expected node executable under bin to be detected")
	}
}

func TestIsPathWithin(t *testing.T) {
	root := filepath.Join("tmp", "vfox", "sdks")

	if !isPathWithin(root, root) {
		t.Fatal("root should be within itself")
	}
	if !isPathWithin(filepath.Join(root, "python", "bin"), root) {
		t.Fatal("child path should be within root")
	}
	if isPathWithin(filepath.Join("tmp", "vfox", "sdks-other"), root) {
		t.Fatal("sibling prefix must not count as child path")
	}
	if isPathWithin(filepath.Join("tmp", "vfox"), root) {
		t.Fatal("parent path must not count as child path")
	}
}

func TestValidateSDKExecutablePath(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "sdk-exe")
	if err := os.WriteFile(exe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "empty", path: "", wantErr: true},
		{name: "missing", path: filepath.Join(tmp, "missing"), wantErr: true},
		{name: "directory", path: tmp, wantErr: true},
		{name: "file", path: exe, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSDKExecutablePath(tt.path)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetCleanedEnvForVfoxRemovesManagedSdkAndShimPaths(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	t.Setenv("VFOX_HOME", tmp)
	userHome := filepath.Join(tmp, "user")
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)

	vfoxSdksDir := filepath.Join(tmp, "sdks")
	vfoxShimDir := filepath.Join(tmp, "path-shims")
	vfoxCacheDir := filepath.Join(tmp, "cache")
	legacySdksDir := filepath.Join(userHome, ".vfox", "sdks")
	legacyCacheDir := filepath.Join(userHome, ".vfox", "cache")
	paths := []string{
		filepath.Join(vfoxSdksDir, "python"),
		vfoxShimDir,
		filepath.Join(vfoxCacheDir, "python", "v-3.14.5", "python-3.14.5"),
		filepath.Join(legacySdksDir, "python", "Scripts"),
		filepath.Join(legacyCacheDir, "nodejs", "v-22.0.0", "node-v22.0.0", "bin"),
		filepath.Join(tmp, "sdks-other"),
		filepath.Join(tmp, "tools"),
	}
	t.Setenv("PATH", strings.Join(paths, string(filepath.ListSeparator)))

	env := app.getCleanedEnvForVfox()
	var gotPath string
	var gotVfoxHome string
	for _, e := range env {
		if strings.HasPrefix(strings.ToLower(e), "path=") {
			gotPath = e[5:]
		}
		if strings.HasPrefix(strings.ToLower(e), "vfox_home=") {
			gotVfoxHome = e[len("VFOX_HOME="):]
		}
	}
	if gotPath == "" {
		t.Fatal("PATH not found in cleaned environment")
	}
	if filepath.Clean(gotVfoxHome) != filepath.Clean(tmp) {
		t.Fatalf("VFOX_HOME = %q, want %q", gotVfoxHome, tmp)
	}

	gotParts := filepath.SplitList(gotPath)
	if len(gotParts) != 2 {
		t.Fatalf("got %d PATH entries, want 2: %v", len(gotParts), gotParts)
	}
	if filepath.Clean(gotParts[0]) != filepath.Clean(filepath.Join(tmp, "sdks-other")) {
		t.Fatalf("unexpected first PATH entry: %q", gotParts[0])
	}
	if filepath.Clean(gotParts[1]) != filepath.Clean(filepath.Join(tmp, "tools")) {
		t.Fatalf("unexpected second PATH entry: %q", gotParts[1])
	}
}

func TestFilterSystemSdksDropsVfoxManagedAndErrorVersions(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	app.setVfoxHome(filepath.Join(tmp, "vfox-home"))
	userHome := filepath.Join(tmp, "user")
	t.Setenv("HOME", userHome)
	t.Setenv("USERPROFILE", userHome)

	input := []SdkInfo{
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "vfox-home", "path-shims", "python.cmd"),
			Versions: []SdkVersion{{Version: "vfoxG: python for python is not available under C:\\vfox-home\\sdks\\python."}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(userHome, ".vfox", "sdks", "python", "python.exe"),
			Versions: []SdkVersion{{Version: "3.14.4"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "vfox-home", "cache", "python", "v-3.14.5", "python-3.14.5", "python.exe"),
			Versions: []SdkVersion{{Version: "3.14.5"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join(tmp, "WindowsApps", "python.exe"),
			Versions: []SdkVersion{{Version: "Python was not found; run without arguments to install from the Microsoft Store."}},
		},
		{
			Name:     "golang",
			Source:   "system",
			Path:     filepath.Join(tmp, "go", "bin", "go.exe"),
			Versions: []SdkVersion{{Version: "1.26.3"}},
		},
	}

	got := app.filterSystemSdks(input)
	if len(got) != 1 {
		t.Fatalf("got %d SDKs, want 1: %+v", len(got), got)
	}
	if got[0].Name != "golang" || got[0].Versions[0].Version != "1.26.3" {
		t.Fatalf("unexpected SDK left after filtering: %+v", got[0])
	}
}

func TestExtractVersionAcceptsExecutablePaths(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		exe  string
		want string
	}{
		{name: "python exe path", raw: "Python 3.12.1\r\n", exe: filepath.Join("tmp", "Python312", "python.exe"), want: "3.12.1"},
		{name: "node cmd path", raw: "v24.15.0\n", exe: filepath.Join("tmp", "tools", "node.cmd"), want: "24.15.0"},
		{name: "unix go path", raw: "go version go1.26.3 windows/amd64\n", exe: filepath.Join("/usr", "local", "go", "bin", "go"), want: "1.26.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractVersion(tt.raw, tt.exe); got != tt.want {
				t.Fatalf("extractVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeExecutableName(t *testing.T) {
	tests := []struct {
		name string
		exe  string
		want string
	}{
		{name: "python exe", exe: filepath.Join("C:", "Tools", "Python", "python.exe"), want: "python"},
		{name: "python launcher", exe: filepath.Join("C:", "Tools", "Python", "python3w.exe"), want: "python"},
		{name: "node cmd", exe: filepath.Join("C:", "Tools", "Node", "node.cmd"), want: "node"},
		{name: "nodejs com", exe: filepath.Join("C:", "Tools", "Node", "nodejs.com"), want: "node"},
		{name: "go binary", exe: filepath.Join("/usr", "local", "go", "bin", "go"), want: "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeExecutableName(tt.exe); got != tt.want {
				t.Fatalf("normalizeExecutableName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsOfficialPluginStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "✓", want: true},
		{status: "√", want: true},
		{status: "yes", want: true},
		{status: "true", want: true},
		{status: "✗", want: false},
		{status: "x", want: false},
		{status: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isOfficialPluginStatus(tt.status); got != tt.want {
				t.Fatalf("isOfficialPluginStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestSDKApiRejectsEmptyInputs(t *testing.T) {
	app := NewApp()
	tmp := t.TempDir()
	app.setVfoxHome(filepath.Join(tmp, "home"))

	if _, err := app.UseVersion("", "1.0.0"); err == nil {
		t.Fatal("UseVersion should reject empty plugin name")
	}
	if _, err := app.UnuseVersion(""); err == nil {
		t.Fatal("UnuseVersion should reject empty plugin name")
	}
	if err := app.InstallVersion("python", ""); err == nil {
		t.Fatal("InstallVersion should reject empty version")
	}
	if err := app.UninstallVersion("", "1.0.0"); err == nil {
		t.Fatal("UninstallVersion should reject empty plugin name")
	}
	if _, err := app.GetVersionPath("python", ""); err == nil {
		t.Fatal("GetVersionPath should reject empty version")
	}
	if _, err := app.SearchVersions(""); err == nil {
		t.Fatal("SearchVersions should reject empty plugin name")
	}
	if err := app.AddNonVfoxSdk("", filepath.Join(tmp, "python"), "3.12.0"); err == nil {
		t.Fatal("AddNonVfoxSdk should reject empty plugin name")
	}
	if err := app.RemoveNonVfoxSdk("python", ""); err == nil {
		t.Fatal("RemoveNonVfoxSdk should reject empty path")
	}
	if _, err := app.UseCustomSdk("", filepath.Join(tmp, "python")); err == nil {
		t.Fatal("UseCustomSdk should reject empty plugin name")
	}
	if got := app.DetectSdkPathVersion("", filepath.Join(tmp, "python")); got != "unknown" {
		t.Fatalf("DetectSdkPathVersion empty name = %q, want unknown", got)
	}
	if _, err := app.GetActiveCustomSdk(""); err == nil {
		t.Fatal("GetActiveCustomSdk should reject empty plugin name")
	}
}

func TestFormatSdkEnvironmentExport(t *testing.T) {
	generatedAt := time.Date(2026, 5, 26, 12, 30, 0, 0, time.UTC)
	report := formatSdkEnvironmentExport(sdkEnvironmentExport{
		GeneratedAt: generatedAt,
		Platform: PlatformInfo{
			OS:                  "windows",
			Name:                "Windows",
			CoreOS:              "windows",
			CoreArch:            "x86_64",
			DownloadPath:        filepath.Join("C:", "Users", "tester", "AppData", "Roaming", "vfoxG", "vfox-home"),
			DefaultDownloadPath: filepath.Join("C:", "Users", "tester", "AppData", "Roaming", "vfoxG", "vfox-home"),
		},
		VfoxInPath:   true,
		PathOverride: true,
		VfoxSdks: []sdkEnvironmentVfoxSdk{
			{
				Name: "python",
				Detail: SdkDetail{
					Name:    "python",
					Current: "3.14.4",
					Versions: []SdkVersionDetail{
						{Version: "3.14.4", IsCurrent: true},
					},
				},
				VersionPaths: map[string]string{"3.14.4": filepath.Join("C:", "sdk", "python")},
			},
		},
		SystemSdks: []SdkInfo{
			{
				Name:     "nodejs",
				Source:   "system",
				Path:     filepath.Join("D:", "NodeJS", "node.exe"),
				Versions: []SdkVersion{{Version: "24.15.0"}},
			},
		},
		CustomSdks: map[string][]SdkInfo{
			"golang": {
				{
					Name:     "golang",
					Source:   "system",
					Path:     filepath.Join("D:", "go", "bin", "go.exe"),
					Versions: []SdkVersion{{Version: "1.26.3"}},
				},
			},
		},
	})

	for _, want := range []string{
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
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

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

func TestGetVfoxHomeUsesConfiguredPathBeforeEnvironment(t *testing.T) {
	app := NewApp()
	configured := filepath.Join(t.TempDir(), "configured")
	envHome := filepath.Join(t.TempDir(), "env")
	t.Setenv("VFOX_HOME", envHome)

	app.setVfoxHome(configured)

	if got := app.getVfoxHome(); filepath.Clean(got) != filepath.Clean(configured) {
		t.Fatalf("getVfoxHome() = %q, want configured path %q", got, configured)
	}
}

func TestNormalizeDownloadPathExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := normalizeDownloadPath(filepath.Join("~", "vfox-data"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "vfox-data")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("normalizeDownloadPath() = %q, want %q", got, want)
	}
}

func TestDefaultVfoxHomeUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	got := NewApp().defaultVfoxHome()
	want := filepath.Join(configDir, "vfoxG", "vfox-home")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("defaultVfoxHome() = %q, want %q", got, want)
	}
}

func TestDefaultUserVfoxHomeUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("AppData", configDir)

	got, err := defaultUserVfoxHome()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configDir, "vfoxG", "vfox-home")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("defaultUserVfoxHome() = %q, want %q", got, want)
	}
}

func TestCorePlatformNames(t *testing.T) {
	if coreOSName() == "" {
		t.Fatal("core OS name must not be empty")
	}
	if coreArchName() == "" {
		t.Fatal("core arch name must not be empty")
	}
}

func TestGetVfoxExecutableFromWorkingDirectoryCore(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(tmp, "core", coreOSName(), coreArchName())
	if err := os.MkdirAll(expectedDir, 0755); err != nil {
		t.Fatal(err)
	}
	expectedExe := filepath.Join(expectedDir, getVfoxExeName())
	if err := os.WriteFile(expectedExe, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	coreDir := app.getCoreDir()
	if filepath.Clean(coreDir) != filepath.Clean(expectedDir) {
		t.Skipf("core dir resolved outside test fixture: %s", coreDir)
	}

	got, err := app.getVfoxExecutable()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(expectedExe) {
		t.Fatalf("got %q, want %q", got, expectedExe)
	}
}

func TestGetVfoxExecutableReportsMissingCore(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	coreDir := app.getCoreDir()
	if !strings.HasPrefix(filepath.Clean(coreDir), filepath.Clean(tmp)) {
		t.Skipf("core dir resolved outside test fixture: %s", coreDir)
	}

	_, err = app.getVfoxExecutable()
	if err == nil {
		t.Fatal("expected missing core executable error")
	}
	if !strings.Contains(err.Error(), "vfox core executable not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomSdksToKeep(t *testing.T) {
	sdks := []SdkInfo{
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join("tmp", "python-a", "bin", "python"),
			Versions: []SdkVersion{{Version: "3.12.0"}},
		},
		{
			Name:     "python",
			Source:   "system",
			Path:     filepath.Join("tmp", "python-b", "bin", "python"),
			Versions: []SdkVersion{{Version: "3.13.0"}},
		},
	}

	got, err := customSdksToKeep(sdks, "")
	if err != nil {
		t.Fatalf("empty keep path returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("empty keep path = %+v, want nil", got)
	}

	got, err = customSdksToKeep(sdks, sdks[1].Path)
	if err != nil {
		t.Fatalf("matching keep path returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d SDKs, want 1", len(got))
	}
	if got[0].Path != sdks[1].Path || got[0].Versions[0].Version != "3.13.0" {
		t.Fatalf("kept SDK mismatch: %+v", got[0])
	}

	_, err = customSdksToKeep(sdks, filepath.Join("tmp", "missing", "bin", "python"))
	if err == nil {
		t.Fatal("missing keep path should return an error")
	}
}

func TestRemoveNonVfoxSdkUsesNormalizedPath(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	app.setVfoxHome(filepath.Join(tmp, "home"))

	exeDir := filepath.Join(tmp, "sdk", "bin")
	if err := os.MkdirAll(exeDir, 0755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(exeDir, "python")
	if err := os.WriteFile(exePath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := app.AddNonVfoxSdk("python", exePath, "3.12.0"); err != nil {
		t.Fatal(err)
	}

	equivalentPath := exeDir + string(filepath.Separator) + "." + string(filepath.Separator) + "python"
	if err := app.RemoveNonVfoxSdk("python", equivalentPath); err != nil {
		t.Fatal(err)
	}

	if got := app.GetNonVfoxSdksMap()["python"]; len(got) != 0 {
		t.Fatalf("custom SDK was not removed: %+v", got)
	}
}
