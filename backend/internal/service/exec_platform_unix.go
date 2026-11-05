//go:build !windows

package service

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

const (
	cliBinaryName     = "skm"
	venvCreateCommand = "python3 -m venv .venv"
	pipInstallCommand = ".venv/bin/pip install -r requirements.txt"
)

func lockExecFile(file *os.File, shared bool) error {
	how := syscall.LOCK_EX
	if shared {
		how = syscall.LOCK_SH
	}
	return syscall.Flock(int(file.Fd()), how)
}

func unlockExecFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

func newShellCommand(ctx context.Context, line string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd
}
