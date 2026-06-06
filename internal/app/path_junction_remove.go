package app

import (
	"os"
	"strings"
)

func (a *App) removeJunctionIfExists(linkPath string) {
	linkPath = strings.TrimSpace(linkPath)
	if linkPath == "" {
		return
	}
	_ = os.RemoveAll(linkPath)
}
