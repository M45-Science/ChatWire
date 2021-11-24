// +build windows

package platform

import (
	"os"
	"os/exec"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
}

func CaptureErrorOut(cdesc *os.File) {
}
