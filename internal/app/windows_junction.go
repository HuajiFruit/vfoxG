//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a *App) ensureJunction(linkPath string, target string) error {
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0 {
			existing, readErr := os.Readlink(linkPath)
			if readErr == nil && strings.EqualFold(filepath.Clean(existing), filepath.Clean(target)) {
				return nil
			}
		} else if fi.IsDir() {
			if strings.EqualFold(filepath.Clean(linkPath), filepath.Clean(target)) {
				return nil
			}
		}
	}
	_ = os.RemoveAll(linkPath)
	_ = os.MkdirAll(filepath.Dir(linkPath), 0755)

	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, target)
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mklink /J failed: %v", err)
	}
	return nil
}
