package fact

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/glob"
)

func GetFactorioBinary() string {
	bloc := ""
	if strings.HasPrefix(cfg.Global.PathData.FactorioBinary, "/") {
		//Absolute path
		bloc = cfg.Global.PathData.FactorioBinary
	} else {
		//Relative path
		bloc = cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.FactorioBinary
	}
	return bloc
}

func DoShowLocks(ch string) {
	var startTime = time.Now()

	glob.GuildLock.Lock()
	time.Sleep(time.Microsecond)
	glob.GuildLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GuildLock")

	glob.CMSBufferLock.Lock()
	time.Sleep(time.Microsecond)
	glob.CMSBufferLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": CMSBufferLock")

	glob.PipeLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PipeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PipeLock")

	glob.GameMapLock.Lock()
	time.Sleep(time.Microsecond)
	glob.GameMapLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GameMapLock")

	glob.ModLoadLock.Lock()
	time.Sleep(time.Microsecond)
	glob.ModLoadLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ModLoadLock")

	glob.PausedTicksLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PausedTicksLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PausedTicksLock")

	glob.GametimeLock.Lock()
	time.Sleep(time.Microsecond)
	glob.GametimeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GametimeLock")

	glob.FactIsRunningLock.Lock()
	time.Sleep(time.Microsecond)
	glob.FactIsRunningLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactIsRunningLock")

	glob.FactAutoStartLock.Lock()
	time.Sleep(time.Microsecond)
	glob.FactAutoStartLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactAutoStartLock")

	glob.DoRebootBotLock.Lock()
	time.Sleep(time.Microsecond)
	glob.DoRebootBotLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoRebootBotLock")

	glob.NumPlayersLock.Lock()
	time.Sleep(time.Microsecond)
	glob.NumPlayersLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NumPlayersLock")

	glob.RelaunchThrottleLock.Lock()
	time.Sleep(time.Microsecond)
	glob.RelaunchThrottleLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": RelaunchThrottleLock")

	glob.PlayerListLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PlayerListLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListLock")

	glob.PlayerListWriteLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PlayerListWriteLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListWriteLock")

	glob.NumLoginsLock.Lock()
	time.Sleep(time.Microsecond)
	glob.NumLoginsLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NumLoginsLock")

	glob.FactorioBootedLock.Lock()
	time.Sleep(time.Microsecond)
	glob.FactorioBootedLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactorioBootedLock")

	glob.PlayerListUpdatedLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PlayerListUpdatedLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListUpdatedLock")

	glob.PlayerListDirtyLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PlayerListDirtyLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListDirtyLock")

	glob.PlayerListSeenDirtyLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PlayerListSeenDirtyLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PlayerListSeenDirtyLock")

	glob.RecordPlayersWriteLock.Lock()
	time.Sleep(time.Microsecond)
	glob.RecordPlayersWriteLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": RecordPlayersWriteLock")

	glob.PasswordListLock.Lock()
	time.Sleep(time.Microsecond)
	glob.PasswordListLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PasswordListLock")

	glob.NoResponseCountLock.Lock()
	time.Sleep(time.Microsecond)
	glob.NoResponseCountLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": NoResponseCountLock")

	glob.FactorioLaunchLock.Lock()
	time.Sleep(time.Microsecond)
	glob.FactorioLaunchLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactorioLaunchLock")

	glob.UpdateFactorioLock.Lock()
	time.Sleep(time.Microsecond)
	glob.UpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": UpdateFactorioLock")

	glob.DoUpdateFactorioLock.Lock()
	time.Sleep(time.Microsecond)
	glob.DoUpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoUpdateFactorioLock")

	glob.ManMinutesLock.Lock()
	time.Sleep(time.Microsecond)
	glob.ManMinutesLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ManMinutesLock")

	glob.ConnectPauseLock.Lock()
	time.Sleep(time.Microsecond)
	glob.ConnectPauseLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ConnectPauseLock")

	CMS(ch, "Complete.")
}

func DoExit() {

	//Show stats
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	mm := GetManMinutes()
	botlog.DoLog(fmt.Sprintf("Stats: Man-hours: %.4f, Activity index: %.4f, Uptime: %v", float64(mm)/60.0, float64(mm)/tnow.Sub(glob.Uptime.Round(time.Second)).Minutes(), tnow.Sub(glob.Uptime.Round(time.Second)).String()))

	time.Sleep(3 * time.Second)
	//This kills all loops!
	glob.ServerRunning = false

	botlog.DoLog("Bot closing, load/save db, and waiting for locks...")
	LoadPlayers()
	WritePlayers()

	time.Sleep(1 * time.Second)

	//File locks
	glob.PlayerListWriteLock.Lock()
	glob.RecordPlayersWriteLock.Lock()

	botlog.DoLog("Closing log files.")
	glob.GameLogDesc.Close()
	glob.BotLogDesc.Close()

	_ = os.Remove("cw.lock")
	//Logs are closed, don't report

	if glob.DS != nil {
		glob.DS.Close()
	}

	time.Sleep(5)
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
			botlog.DoLog("CMS: Line too long! Discarding...")
		}
	}

	glob.CMSBufferLock.Unlock()
}

func LogCMS(channel string, text string) {
	botlog.DoLog(text)
	CMS(channel, text)
}
