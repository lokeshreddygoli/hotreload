package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/lokeshreddygoli/hotreload/internal/filter"
)

// ChangeHandler is called whenever a relevant file change is detected.
type ChangeHandler func(path string)

// maxWatches is a conservative cap to avoid exhausting OS inotify limits.
const maxWatches = 4096

// Watcher watches a directory tree for file changes.
type Watcher struct {
	root       string
	fw         *fsnotify.Watcher
	filter     *filter.Filter
	logger     *slog.Logger
	watchCount int
}

// New creates a Watcher rooted at the given directory.
func New(root string, f *filter.Filter, logger *slog.Logger) (*Watcher, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("accessing root directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", abs)
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	w := &Watcher{
		root:   abs,
		fw:     fw,
		filter: f,
		logger: logger,
	}

	if err := w.addDirTree(abs); err != nil {
		fw.Close()
		return nil, err
	}

	return w, nil
}

// addDirTree recursively watches a directory and all non-ignored subdirectories.
func (w *Watcher) addDirTree(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			w.logger.Warn("Walk error", "path", path, "err", err)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && w.filter.ShouldIgnoreDir(path) {
			w.logger.Debug("Skipping directory", "path", path)
			return filepath.SkipDir
		}
		return w.addDir(path)
	})
}

func (w *Watcher) addDir(path string) error {
	if w.watchCount >= maxWatches {
		w.logger.Warn("Watch limit reached, skipping directory", "path", path, "limit", maxWatches)
		return nil
	}
	if err := w.fw.Add(path); err != nil {
		w.logger.Warn("Failed to watch directory", "path", path, "err", err)
		return nil
	}
	w.watchCount++
	w.logger.Debug("Watching directory", "path", path, "total", w.watchCount)
	return nil
}

func (w *Watcher) removeDir(path string) {
	if err := w.fw.Remove(path); err == nil {
		w.watchCount--
		w.logger.Debug("Unwatched directory", "path", path, "total", w.watchCount)
	}
}

// Watch starts the watch loop and calls onChange for every relevant file change.
// Blocks until the watcher is closed.
func (w *Watcher) Watch(onChange ChangeHandler) {
	for {
		select {
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.handleEvent(event, onChange)

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.logger.Error("Watcher error", "err", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event, onChange ChangeHandler) {
	path := event.Name

	// Dynamically watch newly created directories.
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if !w.filter.ShouldIgnoreDir(path) {
				w.logger.Info("New directory detected, watching", "path", path)
				_ = w.addDirTree(path)
			}
		}
	}

	// Clean up removed directories from the watch list.
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		w.removeDir(path)
	}

	if w.filter.ShouldIgnoreFile(path) {
		return
	}
	if !w.filter.IsRelevantFile(path) {
		return
	}

	op := event.Op
	if op.Has(fsnotify.Write) || op.Has(fsnotify.Create) || op.Has(fsnotify.Remove) {
		w.logger.Info("File changed", "path", path, "op", op)
		onChange(path)
	}
}

// Close shuts down the watcher.
func (w *Watcher) Close() error {
	return w.fw.Close()
}

// WatchCount returns the number of currently watched directories.
func (w *Watcher) WatchCount() int {
	return w.watchCount
}
