package filter_test

import (
	"testing"

	"github.com/lokeshreddygoli/hotreload/internal/filter"
)

func TestShouldIgnoreDir(t *testing.T) {
	f := filter.New()

	ignored := []string{
		"/project/.git",
		"/project/node_modules",
		"/project/vendor",
		"/project/.idea",
		"/project/.vscode",
		"/project/__pycache__",
		"/project/.hidden",
	}
	for _, path := range ignored {
		if !f.ShouldIgnoreDir(path) {
			t.Errorf("expected ShouldIgnoreDir(%q) = true", path)
		}
	}

	watched := []string{
		"/project",
		"/project/cmd",
		"/project/internal",
		"/project/pkg",
		"/project/testserver",
	}
	for _, path := range watched {
		if f.ShouldIgnoreDir(path) {
			t.Errorf("expected ShouldIgnoreDir(%q) = false", path)
		}
	}
}

func TestShouldIgnoreFile(t *testing.T) {
	f := filter.New()

	ignored := []string{
		"/project/main.go.swp",
		"/project/main.go.swx",
		"/project/main.go~",
		"/project/#main.go#",
		"/project/.DS_Store",
		"/project/.gitignore",
		"/project/build.tmp",
		"/project/4913",
	}
	for _, path := range ignored {
		if !f.ShouldIgnoreFile(path) {
			t.Errorf("expected ShouldIgnoreFile(%q) = true", path)
		}
	}

	notIgnored := []string{
		"/project/main.go",
		"/project/handler.go",
		"/project/go.mod",
		"/project/go.sum",
		"/project/README.md",
	}
	for _, path := range notIgnored {
		if f.ShouldIgnoreFile(path) {
			t.Errorf("expected ShouldIgnoreFile(%q) = false", path)
		}
	}
}

func TestIsRelevantFile(t *testing.T) {
	f := filter.New()

	relevant := []string{
		"/project/main.go",
		"/project/cmd/root.go",
		"/project/internal/engine/engine.go",
		"/project/go.mod",
		"/project/go.sum",
	}
	for _, path := range relevant {
		if !f.IsRelevantFile(path) {
			t.Errorf("expected IsRelevantFile(%q) = true", path)
		}
	}

	notRelevant := []string{
		"/project/README.md",
		"/project/Makefile",
		"/project/.env",
		"/project/docker-compose.yml",
	}
	for _, path := range notRelevant {
		if f.IsRelevantFile(path) {
			t.Errorf("expected IsRelevantFile(%q) = false", path)
		}
	}
}
