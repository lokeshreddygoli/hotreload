package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Proc represents a managed OS process.
type Proc struct {
	cmd    *exec.Cmd
	mu     sync.Mutex
	done   chan struct{}
	logger *slog.Logger
}

// Start launches a command and streams its output directly to stdout/stderr.
// Platform-specific setup (process groups / Job Objects) is applied via afterStart.
func Start(ctx context.Context, command string, logger *slog.Logger) (*Proc, error) {
	cmd := buildCmd(ctx, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting process: %w", err)
	}

	p := &Proc{
		cmd:    cmd,
		done:   make(chan struct{}),
		logger: logger,
	}

	// Platform-specific post-start setup (Job Objects on Windows, no-op on Unix).
	afterStart(p)

	// Reap in background so Done() closes promptly on exit.
	go func() {
		defer close(p.done)
		_ = cmd.Wait()
	}()

	return p, nil
}

// Done returns a channel closed when the process exits.
func (p *Proc) Done() <-chan struct{} {
	return p.done
}

// ExitCode returns the exit code once the process has exited, or -1 if still running.
func (p *Proc) ExitCode() int {
	select {
	case <-p.done:
		if p.cmd.ProcessState != nil {
			return p.cmd.ProcessState.ExitCode()
		}
		return -1
	default:
		return -1
	}
}

// Wait blocks until the process exits and returns an error for non-zero exit codes.
func (p *Proc) Wait() error {
	<-p.done
	if p.cmd.ProcessState != nil && p.cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("process exited with code %d", p.cmd.ProcessState.ExitCode())
	}
	return nil
}

// Run executes a command synchronously. Returns ctx.Err() if cancelled.
func Run(ctx context.Context, command string, logger *slog.Logger) error {
	logger.Debug("Running command", "cmd", command)

	cmd := buildCmd(ctx, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

// ShellArgs splits a shell command into an argv slice.
// Handles single and double quoting. Exported for testing.
func ShellArgs(command string) []string {
	return shellArgs(command)
}

func shellArgs(command string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
