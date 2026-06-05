//go:build windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWindowsSafeShimName(t *testing.T) {
	if got := windowsSafeShimName(` a:b/c*d?e"f<g>h|i `); got != "a_b_c_d_e_f_g_h_i" {
		t.Fatalf("windowsSafeShimName() = %q", got)
	}
	if got := windowsSafeShimName("   "); got != "sdk" {
		t.Fatalf("empty shim name = %q, want sdk", got)
	}
}

func TestWindowsSDKShimAliases(t *testing.T) {
	aliases := windowsSDKShimAliases("python")
	want := []string{"python", "python3", "pip", "pip3"}
	for _, alias := range want {
		if !containsStringFold(aliases, alias) {
			t.Fatalf("missing alias %q in %v", alias, aliases)
		}
	}

	seen := map[string]bool{}
	for _, alias := range aliases {
		key := strings.ToLower(alias)
		if seen[key] {
			t.Fatalf("duplicate alias %q in %v", alias, aliases)
		}
		seen[key] = true
	}
}

func TestWindowsShimScriptUsesAllPlaceholders(t *testing.T) {
	script := windowsShimScript("python", "pip", `C:\SDK Root`)
	for _, want := range []string{
		`set "SDK_ROOT=C:\SDK Root"`,
		`%SDK_ROOT%\Scripts\pip.exe`,
		`%SDK_ROOT%\bin\pip.cmd`,
		`where "%ALIAS_NAME%"`,
		`if /I not "%%~fI"=="%~f0"`,
		`vfoxG: pip for python is not available`,
		`no fallback pip was found on PATH`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("shim script missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "%!") || strings.Contains(script, "(MISSING)") {
		t.Fatalf("shim script has fmt placeholder leak:\n%s", script)
	}
}

func TestReadWindowsPathOverrideAliases(t *testing.T) {
	tmp := t.TempDir()
	hijackFile := filepath.Join(tmp, "hijacked_paths.json")
	data := `{
		"python": {
			"Version": 2,
			"Aliases": ["python", "python3", "pip", "pip3"]
		},
		"java": {
			"Version": 2,
			"Aliases": [
				{"value": ["java", "javac"], "Count": 2}
			]
		}
	}`
	if err := os.WriteFile(hijackFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		want []string
	}{
		{name: "python", want: []string{"python", "python3", "pip", "pip3"}},
		{name: "java", want: []string{"java", "javac"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := readWindowsPathOverrideAliases(hijackFile, tt.name)
			if !ok {
				t.Fatal("expected aliases to be read")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("aliases = %v, want %v", got, tt.want)
			}
		})
	}
}

func containsStringFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
