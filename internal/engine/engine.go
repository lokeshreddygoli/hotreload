package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lokeshreddygoli/hotreload/internal/debounce"
	"github.com/lokeshreddygoli/hotreload/internal/filter"
	"github.com/lokeshreddygoli/hotreload/internal/process"
	"github.com/lokeshreddygoli/hotreload/internal/watcher"
)

const (
	debounceInterval        = 300 * time.Millisecond
	crashThreshold          = 2 * time.Second
	maxCrashesBeforeBackoff = 3
	crashBackoff            = 5 * time.Second
)

// Config holds the engine configuration.
type Config struct {
	Root     string
	BuildCmd string
	ExecCmd  string
}

// Engine coordinates file watching, building, and process management.
type Engine struct {
	cfg    Config
	logger *slog.Logger
}

// New creates a new Engine.
func New(cfg Config) (*Engine, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &Engine{cfg: cfg, logger: logger}, nil
}

// Run starts the engine and blocks until SIGINT/SIGTERM is received.
func (e *Engine) Run() error {
	e.logger.Info("hotreload starting",
		"root", e.cfg.Root,
		"build", e.cfg.BuildCmd,
		"exec", e.cfg.ExecCmd,
	)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case sig := <-sigCh:
			e.logger.Info("Shutdown signal received", "signal", sig)
			rootCancel()
		case <-rootCtx.Done():
		}
	}()

	f := filter.New()
	w, err := watcher.New(e.cfg.Root, f, e.logger)
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer w.Close()

	e.logger.Info("Watching directory tree", "root", e.cfg.Root, "dirs", w.WatchCount())

	// triggerCh is buffered at 1 — coalesces in-flight triggers.
	triggerCh := make(chan struct{}, 1)

	db := debounce.New(debounceInterval)
	defer db.Stop()

	go w.Watch(func(path string) {
		db.Trigger(func() {
			select {
			case triggerCh <- struct{}{}:
			default:
			}
		})
	})

	// Fire the initial build immediately on startup.
	triggerCh <- struct{}{}

	var (
		mu          sync.Mutex
		cycleCancel context.CancelFunc
		crashCount  int
		generation  int
	)

	cancelCurrent := func() {
		mu.Lock()
		defer mu.Unlock()
		if cycleCancel != nil {
			cycleCancel()
			cycleCancel = nil
		}
	}
	defer cancelCurrent()

	for {
		select {
		case <-rootCtx.Done():
			e.logger.Info("Shutting down...")
			return nil

		case <-triggerCh:
			cancelCurrent()

			mu.Lock()
			currentCrashes := crashCount
			generation++
			gen := generation
			mu.Unlock()

			cycleCtx, cancel := context.WithCancel(rootCtx)
			mu.Lock()
			cycleCancel = cancel
			mu.Unlock()

			go func(ctx context.Context, crashes, myGen int) {
				updated := e.runCycle(ctx, crashes)
				mu.Lock()
				if generation == myGen {
					crashCount = updated
				}
				mu.Unlock()
			}(cycleCtx, currentCrashes, gen)
		}
	}
}

// runCycle runs one build + server lifecycle. Returns updated crash count.
func (e *Engine) runCycle(ctx context.Context, crashCount int) int {
	if crashCount >= maxCrashesBeforeBackoff {
		e.logger.Warn("Crash loop detected, cooling down",
			"consecutive_crashes", crashCount,
			"backoff", crashBackoff,
		)
		select {
		case <-ctx.Done():
			return crashCount
		case <-time.After(crashBackoff):
		}
		crashCount = 0
	}

	// Build phase
	e.logger.Info("⚙  Building...", "cmd", e.cfg.BuildCmd)
	buildStart := time.Now()

	if err := process.Run(ctx, e.cfg.BuildCmd, e.logger); err != nil {
		if ctx.Err() != nil {
			e.logger.Debug("Build cancelled")
			return crashCount
		}
		e.logger.Error("✗  Build failed",
			"err", err,
			"duration", time.Since(buildStart).Round(time.Millisecond),
		)
		return crashCount
	}

	e.logger.Info("✓  Build succeeded", "duration", time.Since(buildStart).Round(time.Millisecond))

	// Run phase
	e.logger.Info("▶  Starting server...", "cmd", e.cfg.ExecCmd)
	serverStart := time.Now()

	srv, err := process.Start(ctx, e.cfg.ExecCmd, e.logger)
	if err != nil {
		if ctx.Err() != nil {
			return crashCount
		}
		e.logger.Error("✗  Failed to start server", "err", err)
		return crashCount + 1
	}

	select {
	case <-ctx.Done():
		srv.Kill()
		return 0

	case <-srv.Done():
		if ctx.Err() != nil {
			srv.Kill()
			return 0
		}

		elapsed := time.Since(serverStart)
		exitCode := srv.ExitCode()

		if exitCode == 0 {
			e.logger.Info("Server exited cleanly", "uptime", elapsed.Round(time.Millisecond))
			return 0
		}

		if elapsed < crashThreshold {
			newCount := crashCount + 1
			e.logger.Error("✗  Server crashed quickly",
				"exit_code", exitCode,
				"uptime", elapsed.Round(time.Millisecond),
				"consecutive_crashes", newCount,
			)
			return newCount
		}

		e.logger.Error("✗  Server exited unexpectedly",
			"exit_code", exitCode,
			"uptime", elapsed.Round(time.Second),
		)
		return 0
	}
}
