// +build linux

package platform

import (
	"log"
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
		log.Println(erra)
	}
	if errb != nil {
		log.Println(errb)
	}
}
