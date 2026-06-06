package parser

import "testing"

func TestParseInstalledSdksOutput(t *testing.T) {
	commandOutput := `All installed sdk versions
├─┬golang
│ └──1.26.3
└─┬java
  ├──25.0.2+10
  └──21.0.2+13`

	installedSdks := InstalledSdks(commandOutput)

	if len(installedSdks) != 2 {
		t.Fatalf("expected 2 SDKs, got %d", len(installedSdks))
	}
	if installedSdks[0].Name != "golang" || installedSdks[0].Source != "vfox" || len(installedSdks[0].Versions) != 1 || installedSdks[0].Versions[0].Version != "1.26.3" {
		t.Errorf("golang: %+v", installedSdks[0])
	}
	if installedSdks[1].Name != "java" || installedSdks[1].Source != "vfox" || len(installedSdks[1].Versions) != 2 {
		t.Errorf("java: %+v", installedSdks[1])
	}
}

func TestParseInstalledSdksOutputSkipsCustomSystemVersions(t *testing.T) {
	commandOutput := `All installed sdk versions
└─┬python
  ├──custom-sys-python
  └──3.13.12`

	installedSdks := InstalledSdks(commandOutput)

	if len(installedSdks) != 1 {
		t.Fatalf("expected 1 SDK, got %d", len(installedSdks))
	}
	if len(installedSdks[0].Versions) != 1 || installedSdks[0].Versions[0].Version != "3.13.12" {
		t.Fatalf("versions = %+v, want only 3.13.12", installedSdks[0].Versions)
	}
}
