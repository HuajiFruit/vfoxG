package app

import "testing"

func TestIsVersionNotReleasedOutput(t *testing.T) {
	line := `plugin [PreInstall] method error: C:\Users\test\vfox-home\plugin\python\hooks\pre_install.lua:10: The current version is not released`
	if !isVersionNotReleasedOutput(line) {
		t.Fatal("expected pre_install release error to be detected")
	}
	if isVersionNotReleasedOutput("failed to install python") {
		t.Fatal("generic install failure should not be treated as unreleased")
	}
}
