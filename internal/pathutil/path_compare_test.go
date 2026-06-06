package pathutil

import (
	"path/filepath"
	"testing"
)

func TestIsPathWithin(t *testing.T) {
	root := filepath.Join("tmp", "vfox", "sdks")

	if !IsPathWithin(root, root) {
		t.Fatal("root should be within itself")
	}
	if !IsPathWithin(filepath.Join(root, "python", "bin"), root) {
		t.Fatal("child path should be within root")
	}
	if IsPathWithin(filepath.Join("tmp", "vfox", "sdks-other"), root) {
		t.Fatal("sibling prefix must not count as child path")
	}
	if IsPathWithin(filepath.Join("tmp", "vfox"), root) {
		t.Fatal("parent path must not count as child path")
	}
}
