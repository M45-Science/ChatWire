package fact

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
)

func DoExit(delay bool) {

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

	if disc.DS != nil {
		disc.DS.Close()
	}

	fmt.Println("Goodbye.")
	if delay {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
	}
	os.Exit(1)
}

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

	disc.GuildLock.Lock()
	time.Sleep(time.Microsecond)
	disc.GuildLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GuildLock")

	disc.CMSBufferLock.Lock()
	time.Sleep(time.Microsecond)
	disc.CMSBufferLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": desc.CMSBufferLock")

	PipeLock.Lock()
	time.Sleep(time.Microsecond)
	PipeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PipeLock")

	GameMapLock.Lock()
	time.Sleep(time.Microsecond)
	GameMapLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GameMapLock")

	ModLoadLock.Lock()
	time.Sleep(time.Microsecond)
	ModLoadLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ModLoadLock")

	PausedTicksLock.Lock()
	time.Sleep(time.Microsecond)
	PausedTicksLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": PausedTicksLock")

	GametimeLock.Lock()
	time.Sleep(time.Microsecond)
	GametimeLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": GametimeLock")

	FactIsRunningLock.Lock()
	time.Sleep(time.Microsecond)
	FactIsRunningLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactIsRunningLock")

	FactAutoStartLock.Lock()
	time.Sleep(time.Microsecond)
	FactAutoStartLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactAutoStartLock")

	glob.DoRebootBotLock.Lock()
	time.Sleep(time.Microsecond)
	glob.DoRebootBotLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoRebootBotLock")

	NumPlayersLock.Lock()
	time.Sleep(time.Microsecond)
	NumPlayersLock.Unlock()
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

	FactorioBootedLock.Lock()
	time.Sleep(time.Microsecond)
	FactorioBootedLock.Unlock()
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

	FactorioLaunchLock.Lock()
	time.Sleep(time.Microsecond)
	FactorioLaunchLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": FactorioLaunchLock")

	UpdateFactorioLock.Lock()
	time.Sleep(time.Microsecond)
	UpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": UpdateFactorioLock")

	DoUpdateFactorioLock.Lock()
	time.Sleep(time.Microsecond)
	DoUpdateFactorioLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": DoUpdateFactorioLock")

	ManMinutesLock.Lock()
	time.Sleep(time.Microsecond)
	ManMinutesLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ManMinutesLock")

	ConnectPauseLock.Lock()
	time.Sleep(time.Microsecond)
	ConnectPauseLock.Unlock()
	CMS(ch, time.Since(startTime).String()+": ConnectPauseLock")

	CMS(ch, "Complete.")
}

func CMS(channel string, text string) {

	//Split at newlines, so we can batch neatly
	lines := strings.Split(text, "\n")

	disc.CMSBufferLock.Lock()

	for _, line := range lines {

		if len(line) <= 2000 {
			var item disc.CMSBuf
			item.Channel = channel
			item.Text = line

			disc.CMSBuffer = append(disc.CMSBuffer, item)
		} else {
			botlog.DoLog("CMS: Line too long! Discarding...")
		}
	}

	disc.CMSBufferLock.Unlock()
}

func LogCMS(channel string, text string) {
	botlog.DoLog(text)
	CMS(channel, text)
}
