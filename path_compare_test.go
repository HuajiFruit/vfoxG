package main

import (
	"path/filepath"
	"testing"
)

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
