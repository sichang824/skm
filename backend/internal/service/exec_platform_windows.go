package service

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	cliBinaryName     = "skm.exe"
	venvCreateCommand = "python -m venv .venv"
	pipInstallCommand = ".venv/Scripts/pip.exe install -r requirements.txt"
)

func lockExecFile(file *os.File, shared bool) error {
	var flags uint32
	if !shared {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	return windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, &windows.Overlapped{})
}

func unlockExecFile(file *os.File) error {
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{})
}

// Skill manifests use POSIX shell syntax on every platform. Windows users
// running skill commands need Git for Windows' sh.exe on PATH.
func newShellCommand(ctx context.Context, line string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		kill := exec.Command("taskkill.exe", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return cmd
}
