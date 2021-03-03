// +build linux

package platform

import (
	"os"
	"os/exec"
	"syscall"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func CaptureErrorOut(cdesc *os.File) {
	syscall.Dup2(int(cdesc.Fd()), 1)
	syscall.Dup2(int(cdesc.Fd()), 2)
}
