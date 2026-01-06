package cwlog

import (
	"ChatWire/glob"
	"ChatWire/worker"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

/* Normal CW log */
func DoLogCW(format string, args ...interface{}) {
	if glob.CWLogDesc == nil {
		return
	}

	ctime := time.Now()
	_, filename, line, _ := runtime.Caller(1)

	var text string
	if len(args) == 0 {
		text = format
	} else {
		text = fmt.Sprintf(format, append([]interface{}(nil), args...)...)
	}

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %15v:%5v: %v\n", date, filepath.Base(filename), line, text)
	_, err := glob.CWLogDesc.WriteString(buf)
	if err != nil {
		fmt.Println("DoLog: WriteString failure")
		glob.CWLogDesc = nil
		return
	}
}

/* Audit log for moderator/admin actions */
func DoLogAudit(format string, args ...interface{}) {
	if glob.AuditLogDesc == nil {
		return
	}

	ctime := time.Now()
	_, filename, line, _ := runtime.Caller(1)

	var text string
	if len(args) == 0 {
		text = format
	} else {
		text = fmt.Sprintf(format, append([]interface{}(nil), args...)...)
	}

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %15v:%5v: %v\n", date, filepath.Base(filename), line, text)
	_, err := glob.AuditLogDesc.WriteString(buf)
	if err != nil {
		fmt.Println("DoLogAudit: WriteString failure")
		glob.AuditLogDesc = nil
		return
	}
}

/* Game log */
func DoLogGame(format string, args ...interface{}) {
	if glob.GameLogDesc == nil {
		return
	}

	ctime := time.Now()

	var text string
	if len(args) == 0 {
		text = format
	} else {
		text = fmt.Sprintf(format, append([]interface{}(nil), args...)...)
	}

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %v\n", date, text)
	_, err := glob.GameLogDesc.WriteString(buf)
	if err != nil {
		fmt.Println("DoLogGame: WriteString failure")
		glob.GameLogDesc = nil
		return
	}
}

/* Prep everything for the game log */
func StartGameLog() {
	t := time.Now().UTC()

	/* Create our log file names */
	glob.GameLogName = fmt.Sprintf("log/game-%v-%v-%v.log", t.Day(), t.Month(), t.Year())

	/* Make log directory */
	errr := os.MkdirAll("log", os.ModePerm)
	if errr != nil {
		fmt.Print(errr.Error())
		return
	}

	/* Open log files */
	gdesc, erra := os.OpenFile(glob.GameLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	os.Remove("log/newest.log")
	time.Sleep(100 * time.Millisecond)
	os.Symlink(path.Base(glob.GameLogName), "log/newest.log")
	os.Symlink("../factorio/factorio-current.log", "log/factorio-current.log")
	os.Symlink("../factorio/factorio-previous.log", "log/factorio-previous.log")

	/* Handle file errors */
	if erra != nil {
		fmt.Printf("An error occurred when attempting to create game log. Details: %s", erra)
		return
	}

	if glob.GameLogDesc != nil {
		DoLogGame("Rotating log.")
		glob.GameLogDesc.Close()
	}

	/* Save descriptors, open/closed elsewhere */
	glob.GameLogDesc = gdesc

}

/* Prep everything for the cw log */
func StartCWLog() {

	t := time.Now().UTC()

	/* Create our log file names */
	glob.CWLogName = fmt.Sprintf("audit-log/cw-%v-%v-%v.log", t.Day(), t.Month(), t.Year())

	/* Make log directory */
	errr := os.MkdirAll("audit-log", os.ModePerm)
	if errr != nil {
		fmt.Print(errr.Error())
		return
	}

	/* Open log files */
	bdesc, errb := os.OpenFile(glob.CWLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	/* Handle file errors */
	if errb != nil {
		fmt.Printf("An error occurred when attempting to create cw log. Details: %s", errb)
		return
	}

	if glob.CWLogDesc != nil {
		DoLogCW("Rotating log.")
		glob.CWLogDesc.Close()
	}

	/* Save descriptors, open/closed elsewhere */
	glob.CWLogDesc = bdesc

}

/* Prep everything for the audit log */
func StartAuditLog() {

	t := time.Now().UTC()

	/* Create our log file names */
	oldName := glob.AuditLogName
	glob.AuditLogName = fmt.Sprintf("audit-log/audit-%v-%v-%v.log", t.Day(), t.Month(), t.Year())

	/* Make log directory */
	errr := os.MkdirAll("audit-log", os.ModePerm)
	if errr != nil {
		fmt.Print(errr.Error())
		return
	}

	/* Open log files */
	adesc, errb := os.OpenFile(glob.AuditLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	/* Handle file errors */
	if errb != nil {
		fmt.Printf("An error occurred when attempting to create audit log. Details: %s", errb)
		return
	}

	if glob.AuditLogDesc != nil {
		DoLogAudit("Rotating log.")
		glob.AuditLogDesc.Close()
		if fi, err := os.Stat(oldName); err == nil && fi.Size() == 0 {
			_ = os.Remove(oldName)
		}
	}

	/* Save descriptors, open/closed elsewhere */
	glob.AuditLogDesc = adesc

}

func AutoRotateLogs() {
	//Rotate when date changes
	go func() {
		startDay := time.Now().UTC().Day()
		for {
			currentDay := time.Now().UTC().Day()
			if currentDay != startDay {
				startDay = currentDay
				worker.Submit(func() {
					StartCWLog()
					StartGameLog()
					StartAuditLog()
				})
			}
			time.Sleep(time.Second)
		}
	}()
}
