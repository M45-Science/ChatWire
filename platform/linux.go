// +build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func CaptureErrorOut(cdesc *os.File) {
	erra := syscall.Dup2(int(cdesc.Fd()), 1)
	errb := syscall.Dup2(int(cdesc.Fd()), 2)

	if erra != nil {
		fmt.Println(erra)
	}
	if errb != nil {
		fmt.Println(errb)
	}
}
