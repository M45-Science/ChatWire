package support

import (
	"fmt"
	"os"
)

// ErrorLog logs the error to file and then exits the application with an
// exit code of 1.
func ErrorLog(err error) {
	errorlog, rip := os.OpenFile("error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	
	// If we encounter an error here, something is seriously wrong.
	if rip != nil {
		return
	}
	defer errorlog.Close()
	errorlog.WriteString(fmt.Sprintf("%s\n", err))

	return
}
