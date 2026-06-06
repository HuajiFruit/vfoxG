package app

import "testing"

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
