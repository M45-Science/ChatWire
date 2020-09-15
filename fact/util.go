package fact

import (
	"fmt"
	"os"
	"strings"
	"time"

	"../glob"
	"../logs"
)

func DoExit() {

	//Stolen from $stat
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	mm := GetManMinutes()
	logs.Log(fmt.Sprintf("Stats: Man-hours: %.4f, Activity index: %.4f, Uptime: %v", float64(mm)/60.0, float64(mm)/tnow.Sub(glob.Uptime.Round(time.Second)).Minutes(), tnow.Sub(glob.Uptime.Round(time.Second)).String()))

	logs.Log("Bot closing, load/save db, and waiting for locks...")

	LoadPlayers()
	WritePlayers()

	//File locks
	glob.PlayerListWriteLock.Lock()
	glob.RecordPlayersWriteLock.Lock()

	logs.Log("Closing log files.")
	glob.GameLogDesc.Close()
	glob.BotLogDesc.Close()

	if err := os.Remove("cw.lock"); err != nil {
		logs.Log("Lock file missing???")
	}

	//Wait 30 seconds to clear buffer, then lock buffer
	if glob.CMSBuffer != nil {
		logs.Log("Waiting for CMS buffer to finish, locking CMS buffer, and closing Discord session.")
	}
	for x := 0; glob.CMSBuffer != nil && x < 30; x++ {
		time.Sleep(100 * time.Millisecond)
	}
	logs.LogWithoutEcho("Locking CMS buffer.")
	glob.CMSBufferLock.Lock()
	time.Sleep(2 * time.Second)

	if glob.DS != nil {
		logs.LogWithoutEcho("Closing Discord session and exiting.")
		glob.DS.Close()
	}

	fmt.Println("Goodbye.")
	os.Exit(1)
}

func CMS(channel string, text string) {

	//Split at newlines, so we can batch neatly
	lines := strings.Split(text, "\n")

	glob.CMSBufferLock.Lock()

	for _, line := range lines {

		if len(line) <= 2000 {
			var item glob.CMSBuf
			item.Channel = channel
			item.Text = line

			glob.CMSBuffer = append(glob.CMSBuffer, item)
		} else {
			logs.LogWithoutEcho("CMS: Line too long! Discarding...")
		}
	}

	glob.CMSBufferLock.Unlock()
}

func LogCMS(channel string, text string) {
	logs.Log(text)
	CMS(channel, text)
}
