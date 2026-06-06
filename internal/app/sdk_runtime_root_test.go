package app

import (
	"os"
	"path/filepath"
	stdruntime "runtime"
	"testing"
)

func TestSdkRootHasExecutableFindsNestedPython(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "python-3.14.5")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	exeName := "python"
	if stdruntime.GOOS == "windows" {
		exeName = "python.exe"
	}
	if err := os.WriteFile(filepath.Join(root, exeName), []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if !sdkRootHasExecutable(root, "python") {
		t.Fatal("expected python executable to be detected")
	}
}

func TestSdkRootHasExecutableFindsBinExecutable(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	exeName := "node"
	if stdruntime.GOOS == "windows" {
		exeName = "node.exe"
	}
	if err := os.WriteFile(filepath.Join(bin, exeName), []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	if !sdkRootHasExecutable(tmp, "nodejs") {
		t.Fatal("expected node executable under bin to be detected")
	}
}
