//go:build windows

package main

import (
	"os/exec"
)

type execCmd = exec.Cmd

func newExecCmd(name string, args ...string) *execCmd {
	return exec.Command(name, args...)
}

// setSysProcAttr sets Windows-specific process attributes for daemon mode.
func setSysProcAttr(cmd *exec.Cmd) {
	// No special attributes needed on Windows
}
