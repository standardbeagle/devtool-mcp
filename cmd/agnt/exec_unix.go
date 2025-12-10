//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

type execCmd = exec.Cmd

func newExecCmd(name string, args ...string) *execCmd {
	// Don't set Setpgid - it's blocked by seccomp in some environments
	// (WSL, Claude Code sandbox). Process group management isn't needed
	// for agnt run since it's for interactive use.
	return exec.Command(name, args...)
}

// setSysProcAttr sets Unix-specific process attributes for daemon mode.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
