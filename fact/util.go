package fact

import (
	"fmt"
	"os"
	"strings"
	"time"

	"../cfg"
	"../glob"
	"../logs"
)

func DoShowLocks(inch string) {
	var startTime = time.Now()

	var ch string
	if inch == "" {
		ch = cfg.Local.ChannelData.LogID
	} else {
		ch = inch
	}

	glob.GuildLock.Lock()
	glob.GuildLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GuildLock")

	glob.CMSBufferLock.Lock()
	glob.CMSBufferLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": CMSBufferLock")

	glob.PipeLock.Lock()
	glob.PipeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PipeLock")

	glob.GameMapLock.Lock()
	glob.GameMapLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GameMapLock")

	glob.ModLoadLock.Lock()
	glob.ModLoadLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ModLoadLock")

	glob.PausedTicksLock.Lock()
	glob.PausedTicksLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PausedTicksLock")

	glob.GametimeLock.Lock()
	glob.GametimeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GametimeLock")

	glob.SaveTimerLock.Lock()
	glob.SaveTimerLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": SaveTimerLock")

	glob.FactIsRunningLock.Lock()
	glob.FactIsRunningLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactIsRunningLock")

	glob.FactAutoStartLock.Lock()
	glob.FactAutoStartLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactAutoStartLock")

	glob.DoRebootBotLock.Lock()
	glob.DoRebootBotLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoRebootBotLock")

	glob.NumPlayersLock.Lock()
	glob.NumPlayersLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NumPlayersLock")

	glob.RelaunchThrottleLock.Lock()
	glob.RelaunchThrottleLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": RelaunchThrottleLock")

	glob.PlayerListLock.Lock()
	glob.PlayerListLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListLock")

	glob.PlayerListWriteLock.Lock()
	glob.PlayerListWriteLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListWriteLock")

	glob.NumLoginsLock.Lock()
	glob.NumLoginsLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NumLoginsLock")

	glob.FactorioBootedLock.Lock()
	glob.FactorioBootedLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactorioBootedLock")

	glob.PlayerListUpdatedLock.Lock()
	glob.PlayerListUpdatedLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListUpdatedLock")

	glob.PlayerListDirtyLock.Lock()
	glob.PlayerListDirtyLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListDirtyLock")

	glob.PlayerListSeenDirtyLock.Lock()
	glob.PlayerListSeenDirtyLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListSeenDirtyLock")

	glob.RecordPlayersWriteLock.Lock()
	glob.RecordPlayersWriteLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": RecordPlayersWriteLock")

	glob.PasswordListLock.Lock()
	glob.PasswordListLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PasswordListLock")

	glob.NoResponseCountLock.Lock()
	glob.NoResponseCountLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NoResponseCountLock")

	glob.FactorioLaunchLock.Lock()
	glob.FactorioLaunchLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactorioLaunchLock")

	glob.UpdateFactorioLock.Lock()
	glob.UpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": UpdateFactorioLock")

	glob.DoUpdateFactorioLock.Lock()
	glob.DoUpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoUpdateFactorioLock")

	glob.ManMinutesLock.Lock()
	glob.ManMinutesLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ManMinutesLock")

	glob.ConnectPauseLock.Lock()
	glob.ConnectPauseLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ConnectPauseLock")

	CMS(ch, "Complete.")
}

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
	for x := 0; glob.CMSBuffer != nil && x < 15; x++ {
		time.Sleep(1 * time.Second)
	}
	logs.LogWithoutEcho("Locking CMS buffer.")
	glob.CMSBufferLock.Lock()

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
