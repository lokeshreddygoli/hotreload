//go:build !windows

package process

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

const gracefulKillTimeout = 3 * time.Second

// buildCmd creates a cmd for Unix with its own process group.
// Setpgid=true means Kill() can signal the whole tree via -pgid.
func buildCmd(ctx context.Context, command string) *exec.Cmd {
	args := shellArgs(command)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// afterStart is a no-op on Unix; process groups are set up at launch time.
func afterStart(_ *Proc) {}

// Kill sends SIGTERM to the entire process group, escalating to SIGKILL
// if the process hasn't exited within gracefulKillTimeout.
func (p *Proc) Kill() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return
	}

	pid := p.cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		p.logger.Debug("Could not get pgid, killing process directly", "pid", pid)
		_ = p.cmd.Process.Kill()
		return
	}

	p.logger.Debug("Sending SIGTERM to process group", "pgid", pgid)
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	select {
	case <-p.done:
		p.logger.Debug("Process exited gracefully after SIGTERM")
		return
	case <-time.After(gracefulKillTimeout):
		p.logger.Debug("Escalating to SIGKILL", "pgid", pgid)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}

	select {
	case <-p.done:
	case <-time.After(1 * time.Second):
		p.logger.Warn("Process may still be running after SIGKILL", "pgid", pgid)
	}
}
