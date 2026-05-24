//go:build linux

package main

import (
	"os"
	"path/filepath"
)

func unixShellProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".profile"), nil
}

func unixShellProfileCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}
}
