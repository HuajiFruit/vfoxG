package main

import (
	"os"
	"path/filepath"
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

func TestGetCleanedEnvForVfoxRemovesOnlyVfoxSdkPaths(t *testing.T) {
	tmp := t.TempDir()
	app := NewApp()
	t.Setenv("VFOX_HOME", tmp)

	vfoxSdksDir := filepath.Join(tmp, "sdks")
	paths := []string{
		filepath.Join(vfoxSdksDir, "python"),
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
