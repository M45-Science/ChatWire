package support

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	"ChatWire/watcher"
	"ChatWire/worker"
)

func linuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func isIdle() bool {
	return !fact.FactIsRunning || !fact.FactorioBooted || fact.PausedTicks > constants.PauseThresh
}

type debounce struct {
	mu    sync.Mutex
	wait  time.Duration
	timer *time.Timer
	fn    func()
}

func newDebounce(wait time.Duration, fn func()) *debounce {
	return &debounce{wait: wait, fn: fn}
}

func (d *debounce) trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.wait, d.fn)
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
		tokens := make(chan struct{}, 5)
		for i := 0; i < cap(tokens); i++ {
			tokens <- struct{}{}
		}
		refill := time.NewTicker(5 * time.Second)
		defer refill.Stop()
		go func() {
			for range refill.C {
				for len(tokens) < cap(tokens) {
					tokens <- struct{}{}
				}
			}
		}()

		for glob.ServerRunning {

			if disc.DS != nil {
				select {
				case first := <-disc.CMSChan:
					lcopy := []disc.CMSBuf{first}
					timer := time.NewTimer(constants.CMSRate)

				collect:
					for {
						select {
						case msg := <-disc.CMSChan:
							lcopy = append(lcopy, msg)
						case <-timer.C:
							break collect
						}
					}
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}

					var factmsg []string
					var moder []string

					/* Put messages into proper lists */
					for _, msg := range lcopy {
						if strings.EqualFold(msg.Channel, cfg.Local.Channel.ChatChannel) {
							factmsg = append(factmsg, msg.Text)
						} else if strings.EqualFold(msg.Channel, cfg.Global.Discord.ReportChannel) {
							moder = append(moder, msg.Text)
						} else {
							<-tokens
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
							<-tokens
							disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
							glob.SetBootMessage(nil)
							glob.ResetUpdateMessage()
							buf = line
						} else {
							buf = buf + "\n" + line
						}
					}
					if buf != "" {
						<-tokens
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
							<-tokens
							disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
							buf = line
						} else {
							buf = buf + "\n" + line
						}
					}
					if buf != "" {
						<-tokens
						disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
					}

					/* Don't send any more messages for a while (throttle) */
					time.Sleep(constants.CMSRestTime)
				case <-time.After(constants.CMSPollRate):
				}
			} else {
				time.Sleep(constants.CMSPollRate)
			}
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
		saveDirty := newDebounce(10*time.Second, func() {
			worker.Submit(func() {
				cwlog.DoLogCW("Database marked dirty, saving.")
				fact.WritePlayers()
			})
		})

		for glob.ServerRunning {
			<-fact.PlayerListDirtySignal()
			glob.PlayerListDirtyLock.Lock()
			wasDirty := glob.PlayerListDirty
			glob.PlayerListDirty = false
			glob.PlayerListDirtyLock.Unlock()
			if wasDirty {
				saveDirty.trigger()
			}
		}
	}()

	/***********************************************************
	 * Save database (less often), if last seen is marked dirty
	 ***********************************************************/
	go func() {
		time.Sleep(time.Minute)
		saveSeenDirty := newDebounce(30*time.Second, func() {
			worker.Submit(func() {
				//cwlog.DoLogCW("Database last seen flagged, saving.")
				fact.WritePlayers()
			})
		})

		for glob.ServerRunning {
			<-fact.PlayerListSeenDirtySignal()
			glob.PlayerListSeenDirtyLock.Lock()
			wasDirty := glob.PlayerListSeenDirty
			glob.PlayerListSeenDirty = false
			glob.PlayerListSeenDirtyLock.Unlock()
			if wasDirty {
				saveSeenDirty.trigger()
			}
		}
	}()

	/************************************
	 * Database file modification watching
	 ************************************/
	go func() {
		filePath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
		reload := newDebounce(time.Second, func() {
			//cwlog.DoLogCW("Database file modified, loading.")
			fact.LoadPlayers(false, false, false)
		})
		watcher.Watch(filePath, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
	}()

	/****************************************
	 * Global config file modification watching
	 ****************************************/
	go func() {
		reload := newDebounce(time.Second, func() {
			if cfg.ReadGCfg() {
				ConfigSoftMod()
				fact.GenerateFactorioConfig()
				fact.DoUpdateChannelName()
			}
		})
		watcher.Watch(constants.CWGlobalConfig, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
	}()

	/****************************************
	 * Local config file modification watching
	 ****************************************/
	go func() {
		reload := newDebounce(time.Second, func() {
			if cfg.ReadLCfg() {
				util.SetTempFilePrefix(cfg.Local.Callsign + "-")
				ConfigSoftMod()
				fact.GenerateFactorioConfig()
				fact.DoUpdateChannelName()
			}
		})
		watcher.Watch(constants.CWLocalConfig, 5*time.Second, &glob.ServerRunning, func() {
			time.Sleep(time.Second)
			reload.trigger()
		})
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

			if !isIdle() {
				disc.UpdateRoleList()

				/* Live update server description */
				if disc.RoleListUpdated {
					ConfigSoftMod()
					fact.GenerateFactorioConfig()
				}
				disc.RoleListUpdated = false
			}
			time.Sleep(time.Duration(cfg.Local.Options.RoleRefreshIntervalSec) * time.Second)
		}
	}()

	/************************************
	 * Reboot if queued, when server empty
	 ************************************/
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning {
			<-ticker.C

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

	/***********************************
	 * Fix lost connection to log files
	 ***********************************/
	go func() {
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning {
			<-ticker.C

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
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for glob.ServerRunning {
				<-ticker.C

				if !cfg.Local.Options.AutoUpdate {
					continue
				}

				cTime := time.Now()
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
			if !isIdle() {
				fact.UpdateChannelName()
				fact.DoUpdateChannelName()
			}

			time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
		}
	}()

	/* Check for expired pauses */
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning {
			glob.PausedLock.Lock()

			if glob.PausedForConnect {

				limit := time.Minute * 3

				now := time.Now()
				if now.Sub(glob.PausedAt) > limit {

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
					now := time.Now()
					if now.Sub(glob.PausedAt) > time.Minute*30 {
						glob.PausedCount = 0
						glob.PausedAt = now
						glob.PausedFor = ""
						glob.PausedConnectAttempt = false
					}
				}
			}
			glob.PausedLock.Unlock()
			<-ticker.C
		}
	}()

	/**********************************
	* Poll online players once in a while
	**********************************/
	go func() {
		for {
			if isIdle() {
				time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
				continue
			}
			//Game booted
			if fact.FactIsRunning && fact.FactorioBooted {

				//Game isn't paused
				if fact.PausedTicks <= constants.PauseThresh {
					fact.WriteFact(glob.OnlineCommand)
				}
			}
			time.Sleep(time.Duration(cfg.Local.Options.PlayerPollIntervalSec) * time.Second)
		}
	}()

	/****************************/
	/* Check for mod update     */
	/****************************/
	//Every 3 hours
	go func() {

		if *glob.ProxyURL != "" {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for glob.ServerRunning {
				<-ticker.C
				if !cfg.Local.Options.ModUpdate {
					continue
				}

				cTime := time.Now()
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
			if isIdle() {
				time.Sleep(time.Minute)
				continue
			}
			now := time.Now()
			glob.PlayerListLock.Lock() //Lock
			for _, p := range glob.PlayerList {
				if now.Sub(fact.ExpandTime(p.LastSeen)) <= time.Minute {
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
		for glob.ServerRunning {
			// Run reset checks regardless of Factorio state so scheduled resets
			// can still generate a new map even if the server is down.
			fact.CheckMapReset()
			interval := time.Duration(cfg.Local.Options.MapResetCheckIntervalSec) * time.Second
			if fact.HasResetTime() {
				now := time.Now().UTC()
				until := cfg.Local.Options.NextReset.Sub(now)
				if until <= time.Minute {
					interval = time.Second
				}
			}
			if interval <= 0 {
				interval = time.Minute
			}
			time.Sleep(interval)
		}
	}()

	/****************************/
	/* Auto delete modpack files
	 * at the set expire time
	/****************************/
	go func() {
		for glob.ServerRunning {

			time.Sleep(time.Minute)
			numItems := len(cfg.Local.ModPackList)

			if numItems > 0 {
				changed := false
				kept := make([]cfg.ModPackData, 0, numItems)

				for _, item := range cfg.Local.ModPackList {
					expired := item.Path == "" || time.Since(item.Created) > (constants.ModPackLifeMins*time.Minute)
					if !expired {
						kept = append(kept, item)
						continue
					}
					changed = true
					if item.Path != "" {
						if err := os.Remove(item.Path); err != nil {
							cwlog.DoLogCW("Unable to delete expired modpack!")
						} else {
							cwlog.DoLogCW("Deleted expired modpack: " + item.Path)
						}
					}
				}

				if changed {
					cfg.Local.ModPackList = kept
					cfg.WriteLCfg()
				}
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
