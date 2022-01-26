package fact

import (
	"ChatWire/botlog"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"time"

	"github.com/bwmarrin/discordgo"
)

/************************
 * LOCK HANDLER FUNCTIONS
 *************************/

func GetUpdateWarnCounter() int {
	glob.UpdateWarnCounterLock.Lock()
	temp := glob.UpdateWarnCounter
	glob.UpdateWarnCounterLock.Unlock()

	return temp
}

func SetUpdateWarnCounter(val int) {
	glob.UpdateWarnCounterLock.Lock()
	glob.UpdateWarnCounter = val
	glob.UpdateWarnCounterLock.Unlock()
}

func GetManMinutes() int {
	ManMinutesLock.Lock()
	temp := ManMinutes
	ManMinutesLock.Unlock()

	return temp
}

func SetManMinutes(val int) {
	ManMinutesLock.Lock()
	ManMinutes = val
	ManMinutesLock.Unlock()
}

func GetDoUpdateFactorio() bool {
	DoUpdateFactorioLock.Lock()
	temp := DoUpdateFactorio
	DoUpdateFactorioLock.Unlock()

	return temp
}

func SetDoUpdateFactorio(val bool) {
	DoUpdateFactorioLock.Lock()
	DoUpdateFactorio = val
	DoUpdateFactorioLock.Unlock()
}

func GetNumLogins() int {
	glob.NumLoginsLock.RLock()
	temp := glob.NumLogins
	glob.NumLoginsLock.RUnlock()

	return temp
}

func GetPausedTicks() int {
	PausedTicksLock.RLock()
	temp := PausedTicks
	PausedTicksLock.RUnlock()

	return temp
}

func SetPausedTicks(val int) {
	PausedTicksLock.Lock()
	PausedTicks = val
	PausedTicksLock.Unlock()
}

func SetRelaunchThrottle(val int) {
	glob.RelaunchThrottleLock.Lock()
	glob.RelaunchThrottle = val
	glob.RelaunchThrottleLock.Unlock()
}

func GetRelaunchThrottle() int {
	glob.RelaunchThrottleLock.RLock()
	temp := glob.RelaunchThrottle
	glob.RelaunchThrottleLock.RUnlock()

	return temp
}

func SetFactorioBooted(isbooted bool) {
	FactorioBootedLock.Lock()
	FactorioBooted = isbooted
	if isbooted {
		FactorioBootedAt = time.Now()
	} else {
		FactorioBootedAt = time.Time{}
	}
	FactorioBootedLock.Unlock()

}

func IsFactorioBooted() bool {
	FactorioBootedLock.RLock()
	booted := FactorioBooted
	FactorioBootedLock.RUnlock()

	FactIsRunningLock.RLock()
	running := FactIsRunning
	FactIsRunningLock.RUnlock()

	if booted && running {
		return true
	}

	return false
}

func SetModLoadString(str string) {
	ModLoadLock.Lock()
	ModLoadString = str
	ModLoadLock.Unlock()
}

func AddModLoadString(str string) {
	ModLoadLock.Lock()
	if ModLoadString == constants.Unknown {
		ModLoadString = str
	} else {
		ModLoadString = ModLoadString + ", " + str
	}
	ModLoadLock.Unlock()
}

func GetModLoadString() string {
	ModLoadLock.Lock()
	temp := ModLoadString
	ModLoadLock.Unlock()

	return temp
}

func SetNumPlayers(num int) {
	NumPlayersLock.Lock()
	NumPlayers = num
	NumPlayersLock.Unlock()
}

func GetNumPlayers() int {
	NumPlayersLock.RLock()
	temp := NumPlayers
	NumPlayersLock.RUnlock()

	return temp
}

func SetNoResponseCount(num int) {
	glob.NoResponseCountLock.Lock()
	glob.NoResponseCount = num
	glob.NoResponseCountLock.Unlock()
}

func GetNoResposeCount() int {
	glob.NoResponseCountLock.RLock()
	temp := glob.NoResponseCount
	glob.NoResponseCountLock.RUnlock()

	return temp
}

func IsSetRebootBot() bool {
	glob.DoRebootBotLock.RLock()
	temp := glob.DoRebootBot
	glob.DoRebootBotLock.RUnlock()

	return temp
}

func SetBotReboot(should bool) {
	glob.DoRebootBotLock.Lock()
	glob.DoRebootBot = should
	glob.DoRebootBotLock.Unlock()
}

func IsQueued() bool {
	QueueReloadLock.RLock()
	temp := QueueReload
	QueueReloadLock.RUnlock()

	return temp
}

func SetQueued(queue bool) {
	QueueReloadLock.Lock()
	QueueReload = queue
	QueueReloadLock.Unlock()
}

func SetAutoStart(auto bool) {
	FactAutoStartLock.Lock()
	FactAutoStart = auto
	FactAutoStartLock.Unlock()
}

func IsSetAutoStart() bool {
	FactAutoStartLock.RLock()
	temp := FactAutoStart
	FactAutoStartLock.RUnlock()

	return temp
}

func SetFactRunning(run bool, err bool) {
	FactIsRunningLock.Lock()
	wasrun := FactIsRunning
	FactIsRunning = run
	FactIsRunningLock.Unlock()

	if run && GetNoResposeCount() >= 10 {
		//CMS(cfg.Local.ChannelData.ChatID, "Server now appears to be responding again.")
		botlog.DoLog("Server now appears to be responding again.")
	}
	SetNoResponseCount(0)

	if wasrun != run {
		UpdateChannelName()
		return
	}
}

func IsFactRunning() bool {
	FactIsRunningLock.RLock()
	temp := FactIsRunning
	FactIsRunningLock.RUnlock()

	return temp
}

func GetGuild() *discordgo.Guild {
	disc.GuildLock.RLock()
	defer disc.GuildLock.RUnlock()

	return disc.Guild
}

func GetGuildName() string {
	disc.GuildLock.RLock()
	temp := disc.Guild
	disc.GuildLock.RUnlock()

	if temp == nil {
		return constants.Unknown
	} else {
		return disc.Guildname
	}
}

func GetGameTime() string {
	GametimeLock.RLock()
	temp := Gametime
	GametimeLock.RUnlock()

	return temp
}

func SetGameTime(newtime string) {
	GametimeLock.Lock()
	Gametime = newtime
	GametimeLock.Unlock()
}
