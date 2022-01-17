package botlog

import (
	"ChatWire/glob"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

//Normal bot log
func DoLog(text string) {
	ctime := time.Now()
	_, filename, line, _ := runtime.Caller(1)

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %15v:%5v: %v\n", date, filepath.Base(filename), line, text)
	_, err := glob.BotLogDesc.WriteString(buf)
	if err != nil {
		fmt.Println("DoLog: WriteString failure")
		glob.BotLogDesc.Close()
		glob.BotLogDesc = nil
		return
	}
}

//Game log
func DoLogGame(text string) {
	ctime := time.Now()

	date := fmt.Sprintf("%2v:%2v.%2v", ctime.Hour(), ctime.Minute(), ctime.Second())
	buf := fmt.Sprintf("%v: %v\n", date, text)
	_, err := glob.GameLogDesc.WriteString(buf)
	if err != nil {
		fmt.Println("DoLogGame: WriteString failure")
		glob.GameLogDesc.Close()
		glob.GameLogDesc = nil
		return
	}
}

//Prep everything for the game log
func StartGameLog() {
	t := time.Now()

	//Create our log file names
	glob.GameLogName = fmt.Sprintf("log/game-%v-%v-%v.log", t.Day(), t.Month(), t.Year())

	//Make log directory
	errr := os.MkdirAll("log", os.ModePerm)
	if errr != nil {
		fmt.Print(errr.Error())
		return
	}

	//Open log files
	gdesc, erra := os.OpenFile(glob.GameLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//Handle file errors
	if erra != nil {
		fmt.Printf("An error occurred when attempting to create game log. Details: %s", erra)
		return
	}

	//Save descriptors, open/closed elsewhere
	glob.GameLogDesc = gdesc

}

//Prep everything for the bot log
func StartBotLog() {
	t := time.Now()

	//Create our log file names
	glob.BotLogName = fmt.Sprintf("log/bot-%v-%v-%v.log", t.Day(), t.Month(), t.Year())

	//Make log directory
	errr := os.MkdirAll("log", os.ModePerm)
	if errr != nil {
		fmt.Print(errr.Error())
		return
	}

	//Open log files
	bdesc, errb := os.OpenFile(glob.BotLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//Handle file errors
	if errb != nil {
		fmt.Printf("An error occurred when attempting to create bot log. Details: %s", errb)
		return
	}

	//Save descriptors, open/closed elsewhere
	glob.BotLogDesc = bdesc
}
