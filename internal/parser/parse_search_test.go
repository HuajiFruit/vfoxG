package parser

import "testing"

func TestParseSearchVersionsOutput(t *testing.T) {
	commandOutput := `Available versions:
 - 26.2.0 [npm 11.13.0]
 - 24.16.0 (LTS) [npm 11.13.0]
 - 24.16.0 (LTS) [npm 11.13.0]
 - 3.45.0-0.1.pre (beta) [dart 3.13.0]
Use 'vfox install nodejs@<version>'`

	got := SearchVersions(commandOutput)
	want := []string{"26.2.0", "24.16.0", "3.45.0-0.1.pre"}
	if len(got) != len(want) {
		t.Fatalf("SearchVersions() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SearchVersions()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
