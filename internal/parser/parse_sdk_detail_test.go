package parser

import "testing"

func TestParseSdkDetailVersionLine(t *testing.T) {
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

func TestParseSdkDetailOutputUsesSingleCurrentVersion(t *testing.T) {
	commandOutput := `-> 3.14.4 <-- current
-> 3.13.12 <-- current`

	detail := SdkDetail("python", "3.13.12", commandOutput)
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
	commandOutput := `-> 3.14.4 <-- current
-> 3.13.12 <-- current`

	detail := SdkDetail("python", "", commandOutput)
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
	commandOutput := `3.14.4
-> 3.13.12 <-- current`

	detail := SdkDetail("python", "", commandOutput)
	if detail.Current != "3.13.12" {
		t.Fatalf("current = %q, want 3.13.12", detail.Current)
	}
	if !detail.Versions[1].IsCurrent {
		t.Fatal("marked version should be current")
	}
}
