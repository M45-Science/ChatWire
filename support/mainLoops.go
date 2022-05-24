package support

import (
	"fmt"
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
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
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
		for glob.ServerRunning {

			time.Sleep(constants.WatchdogInterval)

			/* Check for updates */
			if !fact.FactIsRunning &&
				(fact.QueueReload || glob.DoRebootCW || fact.DoUpdateFactorio) {
				if time.Since(glob.Uptime) > time.Minute*constants.BootUpdateDelayMin && fact.DoUpdateFactorio {
					fact.FactUpdate()
				}
				fact.DoExit(false)
				return

				/* We are running normally */
			} else if fact.FactIsRunning && fact.FactorioBooted {

				nores := 0
				if fact.PausedTicks <= constants.PauseThresh {

					glob.NoResponseCount = glob.NoResponseCount + 1
					nores = glob.NoResponseCount

					fact.WriteFact("/time")
				}
				if nores == 120 {
					msg := "Factorio unresponsive for over two minutes... rebooting."
					fact.LogCMS(cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					fact.QuitFactorio(msg)

					fact.WaitFactQuit()
					fact.FactorioBooted = false
					fact.SetFactRunning(false)
				}

				/* We aren't running, but should be! */
			} else if !fact.FactIsRunning && !fact.FactorioBooted && fact.FactAutoStart && !fact.DoUpdateFactorio {
				/* Don't relaunch if we are set to auto update */
				launchFactorio()
			}
		}
	}()

	/********************************
	 * Look for lockers
	 ********************************/
	go func() {
		for glob.ServerRunning {
			time.Sleep(time.Millisecond * 100)

			fact.LockerLock.Lock()

			if fact.LockerStart {
				if time.Since(fact.LockerDetectStart) > time.Second*2 && fact.LastLockerName != "" {
					fact.LockerDetectStart = time.Now()
					fact.LockerStart = false

					msg := "Locker bug detected (" + fact.LastLockerName + "), kicking."
					cwlog.DoLogCW(msg)
					fact.WriteFact("/chat " + msg)
					fact.CMS(cfg.Local.Channel.ChatChannel, msg)
					fact.WriteFact("/kick " + fact.LastLockerName)
				}
			}

			fact.LockerLock.Unlock()
		}
	}()

	/********************************
	 * Decrement player suspicion
	 ********************************/
	go func() {
		for glob.ServerRunning {
			time.Sleep(time.Second * 5)

			glob.PlayerSusLock.Lock()

			if len(glob.PlayerSus) > 0 {
				for pname := range glob.PlayerSus {
					if glob.PlayerSus[pname] > 0 {
						glob.PlayerSus[pname]--
					}
				}
			}

			glob.PlayerSusLock.Unlock()
		}
	}()

	/********************************
	 * Watch ban file
	 ********************************/
	go banlist.WatchBanFile()

	/********************************
	 *  Send buffered messages to Discord, batched.
	 ********************************/
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
							if msg.Channel == cfg.Local.Channel.ChatChannel {
								factmsg = append(factmsg, msg.Text)
							} else if msg.Channel == cfg.Global.Discord.ReportChannel {
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
							if oldlen+addlen >= 2000 {
								disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
								buf = line
							} else {
								buf = buf + "\n" + line
							}
						}
						if buf != "" {
							disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
						}

						/* Moderation */
						buf = ""
						for _, line := range moder {
							oldlen := len(buf) + 1
							addlen := len(line)
							if oldlen+addlen >= 2000 {
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

	/***********************************
	 * Delete expired registration codes
	 ***********************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(1 * time.Minute)

			t := time.Now()

			glob.PasswordListLock.Lock()
			for _, pass := range glob.PassList {
				if (t.Unix() - pass.Time) > 300 {
					cwlog.DoLogCW("Invalidating unused registration code for player: " + disc.GetNameFromID(pass.DiscID, false))
					delete(glob.PassList, pass.DiscID)
				}
			}
			glob.PasswordListLock.Unlock()
		}
	}()

	/****************************************
	 * Save vote-rewind data async
	 ****************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(5 * time.Second)

			glob.VoteBoxLock.Lock()

			/* Save if dirty */
			if glob.VoteBox.Dirty {
				fact.WriteRewindVotes()
				glob.VoteBox.Dirty = false
			}
			glob.VoteBoxLock.Unlock()
		}
	}()

	/*****************************************************
	 * Slow-connect, helps players catch up on large maps
	 *****************************************************/
	go func() {
		for glob.ServerRunning {

			time.Sleep(5 * time.Second)
			tn := time.Now()

			if cfg.Local.Options.SoftModOptions.SlowConnect.Enabled {

				fact.SlowConnectLock.Lock()

				if fact.SlowConnectTimer > 0 {
					if tn.Unix()-fact.SlowConnectTimer >= 30 {
						fact.SlowConnectTimer = 0
						fact.SlowConnectEvents = 0

						buf := "Catch-up taking over 30 seconds, returning to normal speed."
						fact.CMS(cfg.Local.Channel.ChatChannel, buf)
						fact.WriteFact("/chat (SYSTEM) " + buf)

						if cfg.Local.Options.SoftModOptions.SlowConnect.Speed > 0.0 {
							fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.Options.SoftModOptions.SlowConnect.Speed))
						} else {
							fact.WriteFact("/gspeed 1.0")
						}
					}
				}

				fact.SlowConnectLock.Unlock()

			}
		}
	}()
	/********************************
	 * Save database, if marked dirty
	 ********************************/
	go func() {
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
		for glob.ServerRunning {
			time.Sleep(5 * time.Minute)
			glob.PlayerListSeenDirtyLock.Lock()

			if glob.PlayerListSeenDirty {
				glob.PlayerListSeenDirty = false

				/* Prevent recursive lock */
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

	/* Read database, if the file was modifed */
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
				fact.LoadPlayers()

				/* Sleep after reading */
				time.Sleep(5 * time.Second)
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
					cwlog.DoLogCW(fmt.Sprintf("Was unable to get guild data from GuildID: %s", err))

					break
				}
				if nguild == nil || err != nil {
					disc.Guildname = constants.Unknown
					cwlog.DoLogCW("Guild data came back nil.")
					break
				} else {

					/* Guild found, exit loop */
					disc.Guild = nguild
					disc.Guildname = nguild.Name
					cwlog.DoLogCW("Guild data linked.")
				}
			}

			/* Update role IDs */
			if disc.Guild != nil {
				changed := false
				for _, role := range disc.Guild.Roles {
					if cfg.Global.Discord.Roles.Moderator != "" &&
						role.Name == cfg.Global.Discord.Roles.Moderator &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Moderator != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Moderator = role.ID
						changed = true

					} else if cfg.Global.Discord.Roles.Regular != "" &&
						role.Name == cfg.Global.Discord.Roles.Regular &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Regular != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Regular = role.ID
						changed = true

					} else if cfg.Global.Discord.Roles.Member != "" &&
						role.Name == cfg.Global.Discord.Roles.Member &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Member != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Member = role.ID
						changed = true

					} else if cfg.Global.Discord.Roles.New != "" &&
						role.Name == cfg.Global.Discord.Roles.New &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.New != role.ID {
						cfg.Global.Discord.Roles.RoleCache.New = role.ID
						changed = true
					} else if cfg.Global.Discord.Roles.Patreon != "" &&
						role.Name == cfg.Global.Discord.Roles.Patreon &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Patreon != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Patreon = role.ID
						changed = true
					} else if cfg.Global.Discord.Roles.Nitro != "" &&
						role.Name == cfg.Global.Discord.Roles.Nitro &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Nitro != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Nitro = role.ID
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
		for glob.ServerRunning {
			time.Sleep(time.Minute * 15)
			if fact.FactorioBooted {
				disc.UpdateRoleList()

				if disc.RoleListUpdated && len(disc.RoleList.Patreons) > 0 {
					fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ","))
				}
				if disc.RoleListUpdated && len(disc.RoleList.NitroBooster) > 0 {
					fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ","))
				}

				/* Live update server description */
				if disc.RoleListUpdated {
					/* goroutine, avoid deadlock */
					go fact.GenerateFactorioConfig()
				}
				disc.RoleListUpdated = false
			}
		}
	}()

	/************************************
	 * Reboot if queued, when server empty
	 ************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(2 * time.Second)

			if fact.QueueReload && fact.NumPlayers == 0 && !fact.DoUpdateFactorio {
				if fact.FactIsRunning && fact.FactorioBooted {
					cwlog.DoLogCW("No players currently online, performing scheduled reboot.")
					fact.QuitFactorio("Server rebooting for maintenance.")
					break //We don't need to loop anymore
				}
			}
		}
	}()

	/*******************************************
	 * Bug players if there is an pending update
	 *******************************************/
	go func() {

		for glob.ServerRunning {
			time.Sleep(30 * time.Second)

			if cfg.Local.Options.AutoUpdate {
				if fact.FactIsRunning && fact.FactorioBooted && fact.NewVersion != constants.Unknown {
					if fact.NumPlayers > 0 {

						/* Warn players */
						if glob.UpdateWarnCounter < glob.UpdateGraceMinutes {
							msg := fmt.Sprintf("(SYSTEM) Factorio update waiting (%v), please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!", fact.NewVersion, glob.UpdateGraceMinutes-glob.UpdateWarnCounter)
							fact.CMS(cfg.Local.Channel.ChatChannel, msg)
							fact.FactChat(fact.AddFactColor("orange", msg))
						}
						time.Sleep(2 * time.Minute)

						/* Reboot anyway */
						if glob.UpdateWarnCounter > glob.UpdateGraceMinutes {
							msg := "(SYSTEM) Rebooting for Factorio update."
							fact.CMS(cfg.Local.Channel.ChatChannel, msg)
							fact.FactChat(fact.AddFactColor("orange", msg))
							glob.UpdateWarnCounter = 0
							fact.QuitFactorio("Rebooting for Factorio update.")
							break /* Stop looping */
						}
						glob.UpdateWarnCounter = (glob.UpdateWarnCounter + 1)
					} else {
						glob.UpdateWarnCounter = 0
						fact.QuitFactorio("Rebooting for Factorio update.")
						break /* Stop looping */
					}
				}
			}
		}
	}()

	/*********************
	 * Check signal files
	 *********************/
	go func() {
		clearOldSignals()
		failureReported := false
		for glob.ServerRunning {

			time.Sleep(10 * time.Second)

			var err error
			var errb error

			/* Queued reboots, regardless of game state */
			if _, err = os.Stat(".queue"); err == nil {
				if errb = os.Remove(".queue"); errb == nil {
					if !fact.QueueReload {
						fact.QueueReload = true
						cwlog.DoLogCW("Reboot queued!")
					}
				} else if errb != nil && !failureReported {
					failureReported = true
					fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .queue file, ignoring.")
				}
			}
			/* Halt, regardless of game state */
			if _, err = os.Stat(".halt"); err == nil {
				if errb = os.Remove(".halt"); errb == nil {
					if fact.FactIsRunning || fact.FactorioBooted {
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "ChatWire is halting, closing Factorio.")
						fact.FactAutoStart = false
						fact.QuitFactorio("Server halted, quitting Factorio.")
						fact.WaitFactQuit()
						fact.DoExit(false)
					} else {
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "ChatWire is halting.")
						fact.DoExit(false)
					}
				} else if errb != nil && !failureReported {
					failureReported = true
					fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .halt file, ignoring.")
				}
			}

			/* Only if game is running */
			if fact.FactIsRunning && fact.FactorioBooted {
				/* Quick reboot */
				if _, err = os.Stat(".qrestart"); err == nil {
					if errb = os.Remove(".qrestart"); errb == nil {
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio quick restarting!")
						fact.QuitFactorio("Server quick restarting...")
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .qrestart file, ignoring.")
					}
				}
				/* Stop game */
				if _, err = os.Stat(".stop"); err == nil {
					if errb = os.Remove(".stop"); errb == nil {
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio stopping!")
						fact.FactAutoStart = false
						fact.QuitFactorio("Server manually stopped.")
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .stop file, ignoring.")
					}
				}
				/* New map */
				if _, err = os.Stat(".newmap"); err == nil {
					if errb = os.Remove(".newmap"); errb == nil {
						fact.Map_reset("", false)
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .stop file, ignoring.")
					}
				}
				/* Message */
				if _, err = os.Stat(".message"); err == nil {
					data, errc := os.ReadFile(".message")
					if errb = os.Remove(".message"); errb == nil {
						if errc == nil && data != nil {
							message := string(data)
							msglen := len(message)
							if msglen > 5 && msglen < 250 {
								message = strings.ReplaceAll(message, "\n", "") /* replace newline */
								message = strings.ReplaceAll(message, "\r", "") /* replace return */
								fact.Map_reset(message, false)
							} else {
								fact.LogCMS(cfg.Local.Channel.ChatChannel, ".message text is invalid, ignoring.")
							}
						}
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .message file, ignoring.")
					}
				}
			} else { /*  Only if game is NOT running */
				/* Start game */
				if _, err = os.Stat(".start"); err == nil {
					if errb = os.Remove(".start"); errb == nil {
						fact.FactAutoStart = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio starting!")
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .start file, ignoring.")
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
			time.Sleep(time.Second * 30)

			var err error
			if _, err = os.Stat(glob.CWLogName); err != nil {

				glob.CWLogDesc.Close()
				glob.CWLogDesc = nil
				cwlog.StartCWLog()
				cwlog.DoLogCW("CWLog file was deleted, recreated.")
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
	go func() {
		for glob.ServerRunning {
			time.Sleep(time.Hour * 3)
			fact.CheckFactUpdate(false)

		}
	}()

	/****************************
	* Refresh channel names
	****************************/
	go func() {

		time.Sleep(time.Second * 15)
		for glob.ServerRunning {
			fact.UpdateChannelName()

			disc.UpdateChannelLock.Lock()
			chname := disc.NewChanName
			oldchname := disc.OldChanName
			disc.UpdateChannelLock.Unlock()

			if oldchname != chname {
				fact.DoUpdateChannelName(false)
			}

			time.Sleep(time.Second * 10)
		}
	}()

	/****************************/
	/* Check for mod update     */
	/****************************/
	go func() {
		time.Sleep(time.Minute)
		for glob.ServerRunning {
			modupdate.CheckMods(false, false)

			time.Sleep(time.Hour * 3)
		}
	}()

}
