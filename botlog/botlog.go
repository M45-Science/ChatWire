package botlog

import (
	"ChatWire/glob"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

func DoLog(text string) {
	ctime := time.Now()
	_, filename, line, _ := runtime.Caller(1)

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %15v:%5v: %v\n", date, filepath.Base(filename), line, text)
	glob.BotLogDesc.WriteString(buf)
}

func DoLogGame(text string) {
	ctime := time.Now()

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %v\n", date, text)
	glob.GameLogDesc.WriteString(buf)
}
