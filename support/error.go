package support

import (
	"fmt"
	"os"
)

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

func Log(text string)  {
	log, rip := os.OpenFile("cord.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	// If we encounter an error here, something is seriously wrong.
	if rip != nil {
		return
	}
	defer log.Close()
	log.WriteString(fmt.Sprintf("%s\n", text))

	return
}