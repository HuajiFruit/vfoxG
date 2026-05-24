//go:build !windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUnixManagedPathBlockQuotesAndPreservesPath(t *testing.T) {
	block := unixManagedPathBlock("vfoxG SDK PATH py'thon", []string{
		"/opt/python/bin",
		"/Applications/SDK Tools/bin",
		"",
	})

	if !strings.Contains(block, "# >>> vfoxG SDK PATH py'thon >>>") {
		t.Fatalf("missing start marker: %s", block)
	}
	if !strings.Contains(block, "'/opt/python/bin'") {
		t.Fatalf("missing quoted unix path: %s", block)
	}
	if !strings.Contains(block, "'/Applications/SDK Tools/bin'") {
		t.Fatalf("path with spaces must be quoted: %s", block)
	}
	if !strings.Contains(block, `:"$PATH"`) {
		t.Fatalf("block must preserve existing PATH at the end: %s", block)
	}
}

func TestUnixRemoveManagedBlockFromString(t *testing.T) {
	original := strings.Join([]string{
		"export KEEP=1",
		unixManagedPathBlock("vfoxG PATH", []string{"/opt/vfox"}),
		"export AFTER=1",
	}, "\n")

	got := unixRemoveManagedBlockFromString(original, "vfoxG PATH")
	if strings.Contains(got, "vfoxG PATH") || strings.Contains(got, "/opt/vfox") {
		t.Fatalf("managed block was not removed: %s", got)
	}
	if !strings.Contains(got, "export KEEP=1") || !strings.Contains(got, "export AFTER=1") {
		t.Fatalf("unrelated profile lines should be preserved: %s", got)
	}
}

func TestUnixRemoveManagedBlockLeavesBrokenBlockUnchanged(t *testing.T) {
	data := "# >>> vfoxG PATH >>>\nexport PATH='/opt/vfox':\"$PATH\"\n"
	got := unixRemoveManagedBlockFromString(data, "vfoxG PATH")
	if got != data {
		t.Fatalf("unterminated block should be left untouched: got %q want %q", got, data)
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote(filepath.Join("/tmp", "Bob's SDK", "bin"))
	if got != "'/tmp/Bob'\\''s SDK/bin'" {
		t.Fatalf("unexpected quoted path: %q", got)
	}
}
