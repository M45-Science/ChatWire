package support

import (
	"os/exec"
	"syscall"
)

func linuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func isProcessRunning(cmd *exec.Cmd) bool {
	// Check if the process state is nil (still running)
	if cmd.ProcessState == nil {
		return true
	}

	// Check if the process has exited
	return !cmd.ProcessState.Exited()
}

func isProcessAlive(pid int) bool {
	// Send signal 0 to the process
	err := syscall.Kill(pid, 0)
	return err == nil
}
