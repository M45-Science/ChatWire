// +build windows

package platform

import (
	"os"
	"os/exec"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
	return
}

func CaptureErrorOut(cdesc *os.File) {
	return
}
