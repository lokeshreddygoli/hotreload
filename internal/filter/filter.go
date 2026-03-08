package filter

import (
	"path/filepath"
	"strings"
)

// ignoredDirs are directory names that should never be watched.
var ignoredDirs = map[string]bool{
	".git":          true,
	".hg":           true,
	".svn":          true,
	"node_modules":  true,
	"vendor":        true,
	".idea":         true,
	".vscode":       true,
	"__pycache__":   true,
	".pytest_cache": true,
	"dist":          true,
	"coverage":      true,
	".nyc_output":   true,
}

// ignoredExts are file extensions that should be ignored.
var ignoredExts = map[string]bool{
	".swp":  true,
	".swx":  true,
	".swo":  true,
	".tmp":  true,
	".temp": true,
	".bak":  true,
	".orig": true,
	".pyc":  true,
	".pyo":  true,
	".class": true,
	".o":    true,
	".a":    true,
	".so":   true,
	".exe":  true,
	".test": true,
}

// Filter decides which paths should be watched and which should be ignored.
type Filter struct{}

// New creates a new Filter.
func New() *Filter {
	return &Filter{}
}

// ShouldIgnoreDir returns true if a directory should be excluded from watching.
func (f *Filter) ShouldIgnoreDir(path string) bool {
	base := filepath.Base(path)

	if strings.HasPrefix(base, ".") && base != "." {
		return true
	}
	return ignoredDirs[base]
}

// ShouldIgnoreFile returns true if a file change event should be ignored.
func (f *Filter) ShouldIgnoreFile(path string) bool {
	base := filepath.Base(path)

	if strings.HasPrefix(base, ".") {
		return true
	}
	if strings.HasSuffix(base, "~") {
		return true
	}
	if strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#") {
		return true
	}
	if base == "4913" {
		return true
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ignoredExts[ext]
}

// IsRelevantFile returns true if the file should trigger a rebuild.
func (f *Filter) IsRelevantFile(path string) bool {
	base := filepath.Base(path)
	if base == "go.mod" || base == "go.sum" {
		return true
	}
	return strings.ToLower(filepath.Ext(path)) == ".go"
}
