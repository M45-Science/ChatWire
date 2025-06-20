package support

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/util"
)

func linuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

/********************
 * Main threads/loops
 ********************/

func MainLoops() {

	/***************
	 * Game watchdog
	 ***************/
	go func() {
		time.Sleep(time.Second * 1)
		for glob.ServerRunning {

			time.Sleep(constants.WatchdogInterval)

			/* Factorio not running */
			if !fact.FactIsRunning && !fact.DoUpdateFactorio {

				if !fact.FactorioBooted && !fact.FactorioBootedAt.IsZero() {
					if time.Since(fact.FactorioBootedAt) < time.Minute*2 {
						glob.CrashLoopCount++
					} else {
						glob.CrashLoopCount = 0
					}
					glob.LastCrash = time.Now()
					fact.FactorioBootedAt = time.Time{}
					if glob.CrashLoopCount >= 3 {
						fact.SetAutolaunch(false, true)
						mapName := fact.GameMapName
						if mapName == "" {
							mapName = "<unknown>"
						}
						msg := fmt.Sprintf("%s-%s: %s: Factorio crashed repeatedly during startup while loading. Moderator attention required, auto-start option disabled.",
							cfg.Global.GroupName, cfg.Local.Callsign, cfg.Local.Name)
						cfg.Local.Options.AutoStart = false
						cfg.WriteLCfg()

						disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, msg)
						cwlog.DoLogCW(msg)
					}
				}

				if fact.QueueFactReboot {
					if cfg.Local.Options.AutoStart {
						fact.SetAutolaunch(true, false)
					}
					fact.QueueFactReboot = false

				} else if fact.QueueReboot || glob.DoRebootCW {
					fact.DoExit(false)
					return

				} else if fact.FactAutoStart &&
					!*glob.NoAutoLaunch {

					if WithinHours() {
						launchFactorio()
					}
				}
				/* We are running normally */
			} else if fact.FactIsRunning && fact.FactorioBooted {

				/* If the game isn't paused, check game time */
				nores := 0
				if fact.PausedTicks <= constants.PauseThresh {

					glob.NoResponseCount = glob.NoResponseCount + 1
					nores = glob.NoResponseCount

					fact.WriteFact("/time")
				}
				/* Just in case factorio hangs, bogs down or is flooded */
				if nores == 120 {
					msg := "Factorio unresponsive for over two minutes... rebooting."
					fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					fact.QuitFactorio(msg)
				}
			}
		}
	}()

	/********************************
	 * Watch ban file for changes
	 ********************************/
	go banlist.WatchBanFile()

	/*************************************************
	 *  Send buffered messages to Discord, batched.
	 *************************************************/
	go func() {
		for glob.ServerRunning {

			if disc.DS != nil {

				/* Check if buffer is active */
				active := false
				disc.CMSBufferLock.Lock()
				if disc.CMSBuffer != nil {
					active = true
				}
				disc.CMSBufferLock.Unlock()

				/* If buffer is active, sleep and wait for it to fill up */
				if active {
					time.Sleep(constants.CMSRate)

					/* Waited for buffer to fill up, grab and clear buffers */
					disc.CMSBufferLock.Lock()
					lcopy := disc.CMSBuffer
					disc.CMSBuffer = nil
					disc.CMSBufferLock.Unlock()

					if lcopy != nil {

						var factmsg []string
						var moder []string

						/* Put messages into proper lists */
						for _, msg := range lcopy {
							if strings.EqualFold(msg.Channel, cfg.Local.Channel.ChatChannel) {
								factmsg = append(factmsg, msg.Text)
							} else if strings.EqualFold(msg.Channel, cfg.Global.Discord.ReportChannel) {
								moder = append(moder, msg.Text)
							} else {
								disc.SmartWriteDiscord(msg.Channel, msg.Text)
							}
						}

						/* Send out buffer, split up if needed */
						/* Factorio */
						buf := ""

						for _, line := range factmsg {
							oldlen := len(buf) + 1
							addlen := len(line)
							if oldlen+addlen >= constants.MaxDiscordMsgLen {
								disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
								glob.SetBootMessage(nil)
								glob.ResetUpdateMessage()
								buf = line
							} else {
								buf = buf + "\n" + line
							}
						}
						if buf != "" {
							disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
							glob.SetBootMessage(nil)
							glob.ResetUpdateMessage()
						}

						/* Moderation */
						buf = ""
						for _, line := range moder {
							oldlen := len(buf) + 1
							addlen := len(line)
							if oldlen+addlen >= constants.MaxDiscordMsgLen {
								disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
								buf = line
							} else {
								buf = buf + "\n" + line
							}
						}
						if buf != "" {
							disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
						}
					}

					/* Don't send any more messages for a while (throttle) */
					time.Sleep(constants.CMSRestTime)
				}

			}

			/* Sleep for a moment before checking buffer again */
			time.Sleep(constants.CMSPollRate)
		}
	}()

	/************************************
	 * Delete expired registration codes
	 ************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(1 * time.Minute)

			t := time.Now()

			glob.PasswordListLock.Lock()
			for _, pass := range glob.PassList {
				if (t.Unix() - pass.Time) > constants.PassExpireSec {
					cwlog.DoLogCW("Invalidating unused registration code for player: " + disc.GetNameFromID(pass.DiscID))
					delete(glob.PassList, pass.DiscID)
				}
			}
			glob.PasswordListLock.Unlock()
		}
	}()

	/********************************
	 * Delete expired panel tokens
	 ********************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(1 * time.Minute)

			t := time.Now()

			glob.PanelTokenLock.Lock()
			for k, tok := range glob.PanelTokens {
				if (t.Unix()-tok.Time) > constants.PassExpireSec || (t.Unix()-tok.Orig) > constants.PanelTokenLimitSec {
					delete(glob.PanelTokens, k)
				}
			}
			glob.PanelTokenLock.Unlock()
		}
	}()

	/********************************
	 * Save database, if marked dirty
	 ********************************/
	go func() {
		time.Sleep(time.Minute)

		for glob.ServerRunning {
			time.Sleep(5 * time.Second)

			wasDirty := false

			glob.PlayerListDirtyLock.Lock()

			if glob.PlayerListDirty {
				glob.PlayerListDirty = false
				wasDirty = true
				/* Prevent recursive lock */
				go func() {
					cwlog.DoLogCW("Database marked dirty, saving.")
					fact.WritePlayers()
				}()
			}
			glob.PlayerListDirtyLock.Unlock()

			/* Sleep after saving */
			if wasDirty {
				time.Sleep(10 * time.Second)
			}
		}
	}()

	/***********************************************************
	 * Save database (less often), if last seen is marked dirty
	 ***********************************************************/
	go func() {
		time.Sleep(time.Minute)

		for glob.ServerRunning {
			time.Sleep(5 * time.Minute)
			glob.PlayerListSeenDirtyLock.Lock()

			if glob.PlayerListSeenDirty {
				glob.PlayerListSeenDirty = false

				/* Prevent deadlock */
				go func() {
					//cwlog.DoLogCW("Database last seen flagged, saving.")
					fact.WritePlayers()
				}()
			}
			glob.PlayerListSeenDirtyLock.Unlock()
		}
	}()

	/************************************
	 * Database file modification watching
	 ************************************/
	go fact.WatchDatabaseFile()

	/****************************************
	 * Global config file modification watching
	 ****************************************/
	go cfg.WatchGCfg()

	/****************************************
	 * Local config file modification watching
	 ****************************************/
	go cfg.WatchLCfg()

	/* Read database, if the file was modified */
	go func() {
		updated := false

		for glob.ServerRunning {

			time.Sleep(5 * time.Second)

			/* Detect update */
			glob.PlayerListUpdatedLock.Lock()
			if glob.PlayerListUpdated {
				updated = true
				glob.PlayerListUpdated = false
			}
			glob.PlayerListUpdatedLock.Unlock()

			if updated {
				updated = false

				//cwlog.DoLogCW("Database file modified, loading.")
				fact.LoadPlayers(false, false, false)
			}

		}
	}()

	/* Reload global config if the file was modified */
	go func() {
		updated := false

		for glob.ServerRunning {

			time.Sleep(5 * time.Second)

			glob.GlobalCfgUpdatedLock.Lock()
			if glob.GlobalCfgUpdated {
				updated = true
				glob.GlobalCfgUpdated = false
			}
			glob.GlobalCfgUpdatedLock.Unlock()

			if updated {
				updated = false

				if cfg.ReadGCfg() {
					ConfigSoftMod()
					fact.GenerateFactorioConfig()
					fact.DoUpdateChannelName()
				}
			}

		}
	}()

	/* Reload local config if the file was modified */
	go func() {
		updated := false

		for glob.ServerRunning {

			time.Sleep(5 * time.Second)

			glob.LocalCfgUpdatedLock.Lock()
			if glob.LocalCfgUpdated {
				updated = true
				glob.LocalCfgUpdated = false
			}
			glob.LocalCfgUpdatedLock.Unlock()

			if updated {
				updated = false

				if cfg.ReadLCfg() {
					util.SetTempFilePrefix(cfg.Local.Callsign + "-")
					ConfigSoftMod()
					fact.GenerateFactorioConfig()
					fact.DoUpdateChannelName()
				}
			}

		}
	}()

	/***************************
	 * Get Guild information
	 * Needed for Discord roles
	 ***************************/
	go func() {
		for glob.ServerRunning {

			/* Get guild id, if we need it */

			if disc.Guild == nil && disc.DS != nil {
				var nguild *discordgo.Guild
				var err error

				/*  Attempt to get the guild from the state,
				 *  If there is an error, fall back to the restapi. */
				nguild, err = disc.DS.State.Guild(cfg.Global.Discord.Guild)
				if err != nil {
					nguild, err = disc.DS.Guild(cfg.Global.Discord.Guild)
					if err != nil {
						cwlog.DoLogCW("Failed to get valid guild data, giving up.")
						break
					}
				}

				if err != nil {
					cwlog.DoLogCW("Was unable to get guild data from GuildID: %s", err)

					break
				}
				if nguild == nil {
					disc.Guildname = constants.Unknown
					cwlog.DoLogCW("Guild data came back nil.")
					break
				} else {

					/* Guild found, exit loop */
					disc.Guild = nguild
					disc.Guildname = nguild.Name
					cwlog.DoLogCW("Guild data linked.")
					fact.LoadPlayers(true, false, false)
				}
			}

			/* Update role IDs */
			if disc.Guild != nil {
				roleMap := buildRoleMap()

				changed := false
				for _, role := range disc.Guild.Roles {
					if updateRoleCache(role, roleMap) {
						changed = true
					}
				}

				if changed {
					cwlog.DoLogCW("Role IDs updated.")
					cfg.WriteGCfg()
				}
			}

			time.Sleep(time.Minute)
		}
	}()

	/*******************************
	 * Update patreon/nitro players
	 *******************************/
	go func() {
		time.Sleep(time.Minute)
		for glob.ServerRunning {

			if fact.FactorioBooted {
				disc.UpdateRoleList()

				/* Live update server description */
				if disc.RoleListUpdated {
					ConfigSoftMod()
					fact.GenerateFactorioConfig()
				}
				disc.RoleListUpdated = false
			}
			time.Sleep(time.Minute)
		}
	}()

	/************************************
	 * Reboot if queued, when server empty
	 ************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(5 * time.Second)

			if fact.FactIsRunning && fact.FactorioBooted && fact.NumPlayers == 0 {

				if fact.QueueReboot && !fact.DoUpdateFactorio {
					cwlog.DoLogCW("No players currently online, performing scheduled reboot.")
					fact.QuitFactorio("Server rebooting for maintenance.")
					break //We don't need to loop anymore, rebooting chat wire.

				} else if fact.QueueFactReboot && !fact.DoUpdateFactorio {
					cwlog.DoLogCW("Stopping Factorio for reboot.")
					fact.QuitFactorio("Rebooting Factorio.")
					time.Sleep(time.Minute)

				} else if fact.DoUpdateFactorio {
					cwlog.DoLogCW("Stopping Factorio for update.")
					fact.QuitFactorio("Updating Factorio.")
					time.Sleep(time.Minute)
				}
			}
		}
	}()

	/*******************************************
	 * Bug players if there is an pending update
	 *******************************************/
	go func() {

		for glob.ServerRunning {

			if fact.FactIsRunning && fact.FactorioBooted && fact.DoUpdateFactorio {
				if fact.NumPlayers > 0 {
					/* Warn players */
					if glob.UpdateWarnCounter < glob.UpdateGraceMinutes {
						msg := fmt.Sprintf("(SYSTEM) Factorio update waiting %v. Please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!",
							fact.NewVersion, glob.UpdateGraceMinutes-glob.UpdateWarnCounter)

						if fact.NewVersion == constants.Unknown {
							msg = fmt.Sprintf("(SYSTEM) Factorio update waiting. Please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!",
								glob.UpdateGraceMinutes-glob.UpdateWarnCounter)
						}
						fact.CMS(cfg.Local.Channel.ChatChannel, msg)
						fact.FactChat(fact.AddFactColor("red", msg))
						fact.FactChat(fact.AddFactColor("cyan", msg))
						fact.FactChat(fact.AddFactColor("black", msg))
					}

					/* Reboot anyway */
					if glob.UpdateWarnCounter > glob.UpdateGraceMinutes {
						msg := "(SYSTEM) Rebooting for Factorio update!"
						fact.FactChat(msg)
						glob.UpdateWarnCounter = 0
						fact.QuitFactorio("Rebooting for Factorio update: " + fact.NewVersion)
						time.Sleep(time.Minute * 15)
					}
					glob.UpdateWarnCounter = (glob.UpdateWarnCounter + 1)

					time.Sleep(time.Minute)
				} else {
					glob.UpdateWarnCounter = 0
					fact.QuitFactorio("Rebooting for Factorio update: " + fact.NewVersion)
					time.Sleep(time.Minute * 10)
				}
			} else {
				time.Sleep(time.Second * 5)
			}
		}
	}()

	/*********************
	 * Check signal files
	 *********************/
	go func() {
		util.ClearOldSignals()
		failureReported := false
		for glob.ServerRunning {

			time.Sleep(10 * time.Second)

			var err error
			var errb error

			/* Queued reboots, regardless of game state */
			if _, err = os.Stat(".queue"); err == nil {
				if errb = os.Remove(".queue"); errb == nil {
					if !fact.QueueReboot {
						fact.QueueReboot = true
						cwlog.DoLogCW("Reboot queued!")
					}
				} else if !failureReported {
					failureReported = true
					cwlog.DoLogCW("Failed to remove .queue file, ignoring.")
				}
			}
			/* Only if game is running */
			if fact.FactIsRunning && fact.FactorioBooted {
				/* Stop game */
				if _, err = os.Stat(".stop"); err == nil {
					if errb = os.Remove(".stop"); errb == nil {
						fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "Factorio stopping!")
						fact.SetAutolaunch(false, false)
						fact.QuitFactorio("Server stopping for maintenance.")
					} else if !failureReported {
						failureReported = true
						cwlog.DoLogCW("Failed to remove .stop file, ignoring.")
					}
				}

				/* Restart game */
				if _, err = os.Stat(".rfact"); err == nil {
					if errb = os.Remove(".rfact"); errb == nil {
						fact.QueueFactReboot = true
					} else if !failureReported {
						failureReported = true
						cwlog.DoLogCW("Failed to remove .rfact file, ignoring.")
					}
				}
			} else { /*  Only if game is NOT running */
				/* Start game */
				if _, err = os.Stat(".start"); err == nil {
					if errb = os.Remove(".start"); errb == nil {
						fact.SetAutolaunch(true, false)
						fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "Factorio starting!")
					} else if !failureReported {
						failureReported = true
						cwlog.DoLogCW("Failed to remove .start file, ignoring.")
					}
				}
			}

		}
	}()

	/***********************************
	 * Fix lost connection to log files
	 ***********************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(time.Second * 5)

			var err error
			if _, err = os.Stat(glob.CWLogName); err != nil {

				glob.CWLogDesc.Close()
				glob.CWLogDesc = nil
				cwlog.StartCWLog()
				cwlog.DoLogCW("CWLog file was deleted, recreated.")
			}

			if _, err = os.Stat(glob.AuditLogName); err != nil {

				glob.AuditLogDesc.Close()
				glob.AuditLogDesc = nil
				cwlog.StartAuditLog()
				cwlog.DoLogAudit("Audit log file was deleted, recreated.")
			}

			if _, err = os.Stat(glob.GameLogName); err != nil {
				glob.GameLogDesc.Close()
				glob.GameLogDesc = nil
				cwlog.StartGameLog()
				cwlog.DoLogGame("GameLog file was deleted, recreated.")
			}
		}
	}()

	/****************************
	* Check for Factorio updates
	****************************/
	//Every 30 minutes
	go func() {

		if *glob.ProxyURL != "" {
			ticker := time.NewTicker(1 * time.Second)

			for glob.ServerRunning {
				<-ticker.C

				if !cfg.Local.Options.AutoUpdate {
					continue
				}

				cTime := time.Now()
				if cTime.Second() != 0 {
					continue
				}

				if cTime.Minute() == 15 || cTime.Minute() == 45 {
					checkFactUpdate()
				}

			}
		} else {

			for glob.ServerRunning {
				time.Sleep(time.Minute * 30)
				if cfg.Local.Options.AutoUpdate {
					checkFactUpdate()
				}
				//Add 0 to 5 minutes of sleep
				time.Sleep(time.Microsecond * time.Duration((rand.Float64() * 60000000 * 5.0)))
			}
		}
	}()

	/****************************
	* Refresh channel names
	****************************/
	go func() {

		for glob.ServerRunning {
			fact.UpdateChannelName()
			fact.DoUpdateChannelName()

			time.Sleep(time.Minute * 6)
		}
	}()

	/* Check for expired pauses */
	go func() {
		for glob.ServerRunning {
			glob.PausedLock.Lock()

			if glob.PausedForConnect {

				limit := time.Minute * 3

				if time.Since(glob.PausedAt) > limit {

					fact.WriteFact(
						fmt.Sprintf("/gspeed %0.2f", cfg.Local.Options.Speed))

					if glob.PausedConnectAttempt {
						msg := "Unpausing, " + glob.PausedFor + " did not finish joining within the time limit."
						fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
					}

					glob.PausedForConnect = false
					glob.PausedFor = ""
					glob.PausedConnectAttempt = false
				}
			} else {
				/* Eventually reset timers */
				if glob.PausedCount > 0 {
					if time.Since(glob.PausedAt) > time.Minute*30 {
						glob.PausedCount = 0
						glob.PausedAt = time.Now()
						glob.PausedFor = ""
						glob.PausedConnectAttempt = false
					}
				}
			}
			glob.PausedLock.Unlock()
			time.Sleep(time.Second * 2)
		}
	}()

	/**********************************
	* Poll online players once in a while
	**********************************/
	go func() {
		for {
			//Game booted
			if fact.FactIsRunning && fact.FactorioBooted {

				//Game isn't paused
				if fact.PausedTicks <= constants.PauseThresh {
					fact.WriteFact(glob.OnlineCommand)
				}
			}
			time.Sleep(time.Minute * 5)
		}
	}()

	/****************************/
	/* Check for mod update     */
	/****************************/
	//Every 3 hours
	go func() {

		if *glob.ProxyURL != "" {
			ticker := time.NewTicker(time.Second)

			for glob.ServerRunning {
				<-ticker.C
				if !cfg.Local.Options.ModUpdate {
					continue
				}

				cTime := time.Now()
				if cTime.Second() != 0 {
					continue
				}
				if cTime.Minute() != 0 {
					continue
				}
				if cTime.Hour()%3 == 0 {
					glob.UpdatersLock.Lock()
					modupdate.CheckMods(false, false)
					glob.UpdatersLock.Unlock()
				}
			}
		} else {

			for glob.ServerRunning {
				time.Sleep(time.Hour * 3)

				if cfg.Local.Options.ModUpdate {
					glob.UpdatersLock.Lock()
					modupdate.CheckMods(false, false)
					glob.UpdatersLock.Unlock()
				}

				//Add 0 to 5 minutes of sleep
				time.Sleep(time.Microsecond * time.Duration((rand.Float64() * 60000000 * 5.0)))
			}
		}
	}()

	/****************************/
	/* Update player time       */
	/****************************/
	go func() {
		for glob.ServerRunning {
			glob.PlayerListLock.Lock() //Lock
			for _, p := range glob.PlayerList {
				if time.Since(fact.ExpandTime(p.LastSeen)) <= time.Minute {
					p.Minutes++
				}
			}
			glob.PlayerListLock.Unlock() //Unlock
			time.Sleep(time.Minute)
		}
	}()

	/****************************/
	/* Update time till reset   */
	/****************************/
	go func() {
		ticker := time.NewTicker(time.Minute)

		for glob.ServerRunning {
			<-ticker.C
			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted {
				UpdateDuration()
			}
		}
	}()

	/****************************/
	/* Check for map resets	    */
	/****************************/
	go func() {
		ticker := time.NewTicker(time.Second)

		for glob.ServerRunning {
			<-ticker.C
			if fact.FactIsRunning &&
				fact.FactorioBooted {
				fact.CheckMapReset()
			}
		}
	}()

	/****************************/
	/* Auto delete modpack files
	 * at the set expire time
	/****************************/
	go func() {
		delme := -1

		for glob.ServerRunning {

			time.Sleep(time.Minute)
			numItems := len(cfg.Local.ModPackList)

			if numItems > 0 {
				for i, item := range cfg.Local.ModPackList {
					if item.Path == "" {
						delme = i
						break
					} else if time.Since(item.Created) > (constants.ModPackLifeMins * time.Minute) {
						delme = i
						break
					}
				}
				if delme >= 0 {
					err := os.Remove(cfg.Local.ModPackList[delme].Path)
					if err != nil {
						cwlog.DoLogCW("Unable to delete expired modpack!")
					}

					cwlog.DoLogCW("Deleted expired modpack: " + cfg.Local.ModPackList[delme].Path)
					if numItems > 1 {
						cfg.Local.ModPackList = append(cfg.Local.ModPackList[:delme], cfg.Local.ModPackList[delme+1:]...)
					} else {
						cfg.Local.ModPackList = []cfg.ModPackData{}
					}
				}
				cfg.WriteLCfg()
			}
		}
	}()

	go checkHours()

}

func checkFactUpdate() {
	glob.ResetUpdateMessage()
	_, msg, err, upToDate := factUpdater.DoQuickLatest(false)
	if msg != "" {
		if !err && !upToDate {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Updated", msg, glob.COLOR_CYAN))
			cwlog.DoLogCW(msg)

			newHist := modupdate.ModHistoryItem{InfoItem: true,
				Name: "Factorio Updated", Notes: "To version: " + fact.NewVersion, Date: time.Now()}
			modupdate.AddModHistory(newHist)
		} else if err && !upToDate {
			//glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel , glob.GetUpdateMessage(), "ERROR", msg, glob.COLOR_RED)
			cwlog.DoLogCW(msg)
		}
	}
	glob.ResetUpdateMessage()
}
