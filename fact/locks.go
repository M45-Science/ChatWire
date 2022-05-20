package fact

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
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
	OnlinePlayersLock.Lock()
	NumPlayers = num
	OnlinePlayersLock.Unlock()
}

func GetNumPlayers() int {
	OnlinePlayersLock.RLock()
	temp := NumPlayers
	OnlinePlayersLock.RUnlock()

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

func IsSetRebootCW() bool {
	glob.DoRebootCWLock.RLock()
	temp := glob.DoRebootCW
	glob.DoRebootCWLock.RUnlock()

	return temp
}

func SetCWReboot(should bool) {
	glob.DoRebootCWLock.Lock()
	glob.DoRebootCW = should
	glob.DoRebootCWLock.Unlock()
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
		//CMS(cfg.Local.Channel.ChatChannel, "Server now appears to be responding again.")
		cwlog.DoLogCW("Server now appears to be responding again.")
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
