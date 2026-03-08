//go:build windows

package process

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

const gracefulKillTimeout = 3 * time.Second

// Windows Job Object API constants.
const (
	jobObjectLimitKillOnJobClose    = 0x2000
	jobObjectExtendedLimitInfoClass = 9
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObject          = kernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
)

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectBasicLimitInfo struct {
	PerProcessUserTimeLimit uint64
	PerJobUserTimeLimit     uint64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobObjectExtendedLimitInfo struct {
	BasicLimitInformation jobObjectBasicLimitInfo
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// jobHandles maps each Proc to its Windows Job Object handle.
var jobHandles = make(map[*Proc]syscall.Handle)

// buildCmd creates a cmd for Windows, wrapped in cmd.exe /C so shell
// built-ins and PATH resolution work as expected.
func buildCmd(ctx context.Context, command string) *exec.Cmd {
	args := shellArgs(command)
	cmd := exec.CommandContext(ctx, "cmd.exe", append([]string{"/C"}, args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd
}

// afterStart assigns the new process to a Windows Job Object so the whole
// process tree can be terminated atomically via Kill().
func afterStart(p *Proc) {
	initJobObject(p)
}

// createJobObject creates a Job Object, configures KILL_ON_JOB_CLOSE, and
// assigns the given process handle to it.
func createJobObject(handle syscall.Handle) (syscall.Handle, error) {
	job, _, err := procCreateJobObject.Call(0, 0)
	if job == 0 {
		return 0, fmt.Errorf("CreateJobObject: %w", err)
	}
	jobHandle := syscall.Handle(job)

	info := jobObjectExtendedLimitInfo{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(jobHandle),
		jobObjectExtendedLimitInfoClass,
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		_ = syscall.CloseHandle(jobHandle)
		return 0, fmt.Errorf("SetInformationJobObject: %w", err)
	}

	ret, _, err = procAssignProcessToJobObject.Call(uintptr(jobHandle), uintptr(handle))
	if ret == 0 {
		_ = syscall.CloseHandle(jobHandle)
		return 0, fmt.Errorf("AssignProcessToJobObject: %w", err)
	}

	return jobHandle, nil
}

// initJobObject associates the process with a Job Object immediately after start.
// Falls back gracefully — Kill() will use taskkill if no job is available.
func initJobObject(p *Proc) {
	if p.cmd.Process == nil {
		return
	}

	handle, err := syscall.OpenProcess(
		syscall.PROCESS_ALL_ACCESS, false, uint32(p.cmd.Process.Pid),
	)
	if err != nil {
		p.logger.Debug("OpenProcess failed, job object unavailable", "err", err)
		return
	}

	job, err := createJobObject(handle)
	_ = syscall.CloseHandle(handle)
	if err != nil {
		p.logger.Debug("createJobObject failed", "err", err)
		return
	}

	jobHandles[p] = job
	p.logger.Debug("Process assigned to Job Object", "pid", p.cmd.Process.Pid)

	// Auto-clean job handle when process exits.
	go func() {
		<-p.done
		p.mu.Lock()
		if j, ok := jobHandles[p]; ok {
			_ = syscall.CloseHandle(j)
			delete(jobHandles, p)
		}
		p.mu.Unlock()
	}()
}

// Kill terminates the process tree on Windows.
// Order: Job Object termination → taskkill /F /T → direct Process.Kill.
func (p *Proc) Kill() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return
	}

	pid := p.cmd.Process.Pid

	// Primary: terminate via Job Object (kills entire tree instantly).
	if job, ok := jobHandles[p]; ok {
		p.logger.Debug("Terminating Job Object", "pid", pid)
		ret, _, err := procTerminateJobObject.Call(uintptr(job), 1)
		_ = syscall.CloseHandle(job)
		delete(jobHandles, p)
		if ret != 0 {
			select {
			case <-p.done:
			case <-time.After(gracefulKillTimeout):
				p.logger.Warn("Process did not exit after Job Object termination", "pid", pid)
			}
			return
		}
		p.logger.Debug("TerminateJobObject failed, falling back to taskkill", "err", err)
	}

	// Fallback 1: taskkill /F /T — kills the process tree recursively.
	p.logger.Debug("Running taskkill /F /T", "pid", pid)
	tk := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	if err := tk.Run(); err != nil {
		p.logger.Debug("taskkill failed, using Process.Kill", "err", err)
		// Fallback 2: kills only the root process.
		_ = p.cmd.Process.Kill()
	}

	select {
	case <-p.done:
	case <-time.After(gracefulKillTimeout):
		p.logger.Warn("Process may still be running after taskkill", "pid", pid)
	}
}
