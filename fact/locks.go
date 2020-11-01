package fact

import (
	"time"

	"../config"
	"../constants"
	"../glob"
	"../logs"

	"github.com/bwmarrin/discordgo"
)

//************************
//LOCK HANDLER FUNCTIONS
//************************

func GetManMinutes() int {
	glob.ManMinutesLock.Lock()
	defer glob.ManMinutesLock.Unlock()

	return glob.ManMinutes
}

func SetManMinutes(val int) {
	glob.ManMinutesLock.Lock()
	defer glob.ManMinutesLock.Unlock()

	glob.ManMinutes = val
}

func GetDoUpdateFactorio() bool {
	glob.DoUpdateFactorioLock.Lock()
	defer glob.DoUpdateFactorioLock.Unlock()

	return glob.DoUpdateFactorio
}

func SetDoUpdateFactorio(val bool) {
	glob.DoUpdateFactorioLock.Lock()
	defer glob.DoUpdateFactorioLock.Unlock()

	glob.DoUpdateFactorio = val
}

func GetNumLogins() int {
	glob.NumLoginsLock.RLock()
	defer glob.NumLoginsLock.RUnlock()

	return glob.NumLogins
}

func GetPausedTicks() int {
	glob.PausedTicksLock.RLock()
	defer glob.PausedTicksLock.RUnlock()

	return glob.PausedTicks
}

func SetPausedTicks(val int) {
	glob.PausedTicksLock.Lock()
	defer glob.PausedTicksLock.Unlock()

	glob.PausedTicks = val
}

func SetRelaunchThrottle(val int) {
	glob.RelaunchThrottleLock.Lock()
	defer glob.RelaunchThrottleLock.Unlock()

	glob.RelaunchThrottle = val
}

func GetRelaunchThrottle() int {
	glob.RelaunchThrottleLock.RLock()
	defer glob.RelaunchThrottleLock.RUnlock()

	return glob.RelaunchThrottle
}

func SetFactorioBooted(isbooted bool) {
	glob.FactorioBootedLock.Lock()
	defer glob.FactorioBootedLock.Unlock()

	glob.FactorioBooted = isbooted
}

func IsFactorioBooted() bool {
	glob.FactorioBootedLock.RLock()
	glob.FactIsRunningLock.RLock()
	defer glob.FactorioBootedLock.RUnlock()
	defer glob.FactIsRunningLock.RUnlock()

	if glob.FactorioBooted && glob.FactIsRunning {
		return true
	}

	return false
}

func SetNumPlayers(num int) {
	glob.NumPlayersLock.Lock()
	defer glob.NumPlayersLock.Unlock()

	glob.NumPlayers = num
}

func GetNumPlayers() int {
	glob.NumPlayersLock.RLock()
	defer glob.NumPlayersLock.RUnlock()

	return glob.NumPlayers
}

func SetNoResponseCount(num int) {
	glob.NoResponseCountLock.Lock()
	defer glob.NoResponseCountLock.Unlock()

	glob.NoResponseCount = num
}

func GetNoResposeCount() int {
	glob.NoResponseCountLock.RLock()
	defer glob.NoResponseCountLock.RUnlock()

	return glob.NoResponseCount
}

func IsSetRebootBot() bool {
	glob.DoRebootBotLock.RLock()
	defer glob.DoRebootBotLock.RUnlock()

	return glob.DoRebootBot
}

func SetBotReboot(should bool) {
	glob.DoRebootBotLock.Lock()
	defer glob.DoRebootBotLock.Unlock()

	glob.DoRebootBot = should
}

func IsQueued() bool {
	glob.QueueReloadLock.RLock()
	defer glob.QueueReloadLock.RUnlock()

	return glob.QueueReload
}

func SetQueued(queue bool) {
	glob.QueueReloadLock.Lock()
	defer glob.QueueReloadLock.Unlock()

	glob.QueueReload = queue
}

func SetAutoStart(auto bool) {
	glob.FactAutoStartLock.Lock()
	defer glob.FactAutoStartLock.Unlock()

	glob.FactAutoStart = auto
}

func IsSetAutoStart() bool {
	glob.FactAutoStartLock.RLock()
	defer glob.FactAutoStartLock.RUnlock()

	return glob.FactAutoStart
}

func SetFactRunning(run bool, err bool) {
	glob.FactIsRunningLock.Lock()
	wasrun := glob.FactIsRunning
	glob.FactIsRunning = run
	glob.FactIsRunningLock.Unlock()

	if run == true && GetNoResposeCount() >= 30 {
		CMS(config.Config.FactorioChannelID, "Server now appears to be responding again.")
		logs.Log("Server now appears to be responding again.")
	}
	SetNoResponseCount(0)

	if wasrun != run {
		UpdateChannelName()
		if run == false {
			FactorioIsOffline(err)
		}
		return
	}
}

func IsFactRunning() bool {
	glob.FactIsRunningLock.RLock()
	defer glob.FactIsRunningLock.RUnlock()

	return glob.FactIsRunning
}

func SetSaveTimer() {
	glob.SaveTimerLock.Lock()
	glob.SaveTimer = time.Now()
	glob.SaveTimerLock.Unlock()
}

func GetSaveTimer() time.Time {
	glob.SaveTimerLock.RLock()
	defer glob.SaveTimerLock.RUnlock()

	return glob.SaveTimer
}

func GetGuild() *discordgo.Guild {
	glob.GuildLock.RLock()
	defer glob.GuildLock.RUnlock()

	return glob.Guild
}

func GetGuildName() string {
	glob.GuildLock.RLock()
	defer glob.GuildLock.RUnlock()

	if glob.Guild == nil {
		return constants.Unknown
	} else {
		return glob.Guildname
	}
}

func GetGameTime() string {
	glob.GametimeLock.RLock()
	defer glob.GametimeLock.RUnlock()

	temp := glob.Gametime
	return temp
}

func SetGameTime(newtime string) {
	glob.GametimeLock.Lock()
	defer glob.GametimeLock.Unlock()

	glob.Gametime = newtime
}
