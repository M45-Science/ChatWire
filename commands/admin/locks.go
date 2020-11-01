package admin

import (
	"time"

	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
)

//Archive map
func ShowLocks(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	var startTime = time.Now()

	glob.GuildLock.Lock()
	glob.GuildLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": GuildLock")

	glob.CMSBufferLock.Lock()
	glob.CMSBufferLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": CMSBufferLock")

	glob.PipeLock.Lock()
	glob.PipeLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PipeLock")

	glob.GameMapLock.Lock()
	glob.GameMapLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": GameMapLock")

	glob.ModLoadLock.Lock()
	glob.ModLoadLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": ModLoadLock")

	glob.PausedTicksLock.Lock()
	glob.PausedTicksLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PausedTicksLock")

	glob.GametimeLock.Lock()
	glob.GametimeLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": GametimeLock")

	glob.SaveTimerLock.Lock()
	glob.SaveTimerLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": SaveTimerLock")

	glob.FactIsRunningLock.Lock()
	glob.FactIsRunningLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": FactIsRunningLock")

	glob.FactAutoStartLock.Lock()
	glob.FactAutoStartLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": FactAutoStartLock")

	glob.DoRebootBotLock.Lock()
	glob.DoRebootBotLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": DoRebootBotLock")

	glob.NumPlayersLock.Lock()
	glob.NumPlayersLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": NumPlayersLock")

	glob.RelaunchThrottleLock.Lock()
	glob.RelaunchThrottleLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": RelaunchThrottleLock")

	glob.PlayerListLock.Lock()
	glob.PlayerListLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PlayerListLock")

	glob.PlayerListWriteLock.Lock()
	glob.PlayerListWriteLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PlayerListWriteLock")

	glob.NumLoginsLock.Lock()
	glob.NumLoginsLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": NumLoginsLock")

	glob.FactorioBootedLock.Lock()
	glob.FactorioBootedLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": FactorioBootedLock")

	glob.PlayerListUpdatedLock.Lock()
	glob.PlayerListUpdatedLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PlayerListUpdatedLock")

	glob.PlayerListDirtyLock.Lock()
	glob.PlayerListDirtyLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PlayerListDirtyLock")

	glob.PlayerListSeenDirtyLock.Lock()
	glob.PlayerListSeenDirtyLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PlayerListSeenDirtyLock")

	glob.RecordPlayersWriteLock.Lock()
	glob.RecordPlayersWriteLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": RecordPlayersWriteLock")

	glob.PasswordListLock.Lock()
	glob.PasswordListLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": PasswordListLock")

	glob.NoResponseCountLock.Lock()
	glob.NoResponseCountLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": NoResponseCountLock")

	glob.FactorioLaunchLock.Lock()
	glob.FactorioLaunchLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": FactorioLaunchLock")

	glob.UpdateFactorioLock.Lock()
	glob.UpdateFactorioLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": UpdateFactorioLock")

	glob.DoUpdateFactorioLock.Lock()
	glob.DoUpdateFactorioLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": DoUpdateFactorioLock")

	glob.ManMinutesLock.Lock()
	glob.ManMinutesLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": ManMinutesLock")

	glob.ConnectPauseLock.Lock()
	glob.ConnectPauseLock.Unlock()
	fact.CMS(m.ChannelID, time.Since(startTime).String()+": ConnectPauseLock")

}
