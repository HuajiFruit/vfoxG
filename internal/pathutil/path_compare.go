package pathutil

import (
	"path/filepath"
	stdruntime "runtime"
	"strings"
)

// IsPathWithin reports whether path is root itself or a child of root.
func IsPathWithin(path string, root string) bool {
	path = strings.TrimSpace(path)
	root = strings.TrimSpace(root)
	if path == "" || root == "" {
		return false
	}

	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if stdruntime.GOOS == "windows" {
		path = strings.ToLower(path)
		root = strings.ToLower(root)
	}
	if path == root {
		return true
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func SamePath(leftPath string, rightPath string) bool {
	leftPath = filepath.Clean(strings.TrimSpace(leftPath))
	rightPath = filepath.Clean(strings.TrimSpace(rightPath))
	if stdruntime.GOOS == "windows" {
		return strings.EqualFold(leftPath, rightPath)
	}
	return leftPath == rightPath
}
