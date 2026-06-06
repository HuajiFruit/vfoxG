package main

import (
	"path/filepath"
	"testing"
)

func TestExtractVersionAcceptsExecutablePaths(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		exe  string
		want string
	}{
		{name: "python exe path", raw: "Python 3.12.1\r\n", exe: filepath.Join("tmp", "Python312", "python.exe"), want: "3.12.1"},
		{name: "node cmd path", raw: "v24.15.0\n", exe: filepath.Join("tmp", "tools", "node.cmd"), want: "24.15.0"},
		{name: "unix go path", raw: "go version go1.26.3 windows/amd64\n", exe: filepath.Join("/usr", "local", "go", "bin", "go"), want: "1.26.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractVersion(tt.raw, tt.exe); got != tt.want {
				t.Fatalf("extractVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeExecutableName(t *testing.T) {
	tests := []struct {
		name string
		exe  string
		want string
	}{
		{name: "python exe", exe: filepath.Join("C:", "Tools", "Python", "python.exe"), want: "python"},
		{name: "python launcher", exe: filepath.Join("C:", "Tools", "Python", "python3w.exe"), want: "python"},
		{name: "node cmd", exe: filepath.Join("C:", "Tools", "Node", "node.cmd"), want: "node"},
		{name: "nodejs com", exe: filepath.Join("C:", "Tools", "Node", "nodejs.com"), want: "node"},
		{name: "go binary", exe: filepath.Join("/usr", "local", "go", "bin", "go"), want: "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeExecutableName(tt.exe); got != tt.want {
				t.Fatalf("normalizeExecutableName() = %q, want %q", got, tt.want)
			}
		})
	}
}
