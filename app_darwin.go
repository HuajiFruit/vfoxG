//go:build darwin

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
	return filepath.Join(home, ".zprofile"), nil
}

func unixShellProfileCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".zprofile"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".profile"),
	}
}
