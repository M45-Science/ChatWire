package support

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	embed "github.com/Clinet/discordgo-embed"
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
		time.Sleep(time.Second * 1)
		for glob.ServerRunning {

			time.Sleep(constants.WatchdogInterval)

			/* Factorio not running */
			if !fact.FactIsRunning && !fact.DoUpdateFactorio {

				if fact.QueueFactReboot {
					if cfg.Local.Options.AutoStart {
						fact.FactAutoStart = true
					}
					fact.QueueFactReboot = false

				} else if fact.QueueReboot || glob.DoRebootCW {
					fact.DoExit(false)
					return

				} else if fact.FactAutoStart &&
					!*glob.NoAutoLaunch {

					if WithinHours() {
						time.Sleep(time.Second)
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
					fact.LogCMS(cfg.Local.Channel.ChatChannel, msg)
					glob.RelaunchThrottle = 0
					fact.QuitFactorio(msg)

					fact.WaitFactQuit()
					fact.FactorioBooted = false
					fact.SetFactRunning(false)
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
				fact.LoadPlayers(false, false, false)
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
				changed := false
				/* TODO: Clean up dupe code. This started off simple and grew */
				for _, role := range disc.Guild.Roles {
					if cfg.Global.Discord.Roles.Admin != "" &&
						role.Name == cfg.Global.Discord.Roles.Admin &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Admin != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Admin = role.ID
						changed = true

					} else if cfg.Global.Discord.Roles.Moderator != "" &&
						role.Name == cfg.Global.Discord.Roles.Moderator &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Moderator != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Moderator = role.ID
						changed = true

					} else if cfg.Global.Discord.Roles.Veteran != "" &&
						role.Name == cfg.Global.Discord.Roles.Veteran &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Veteran != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Veteran = role.ID
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
					} else if cfg.Global.Discord.Roles.Supporter != "" &&
						role.Name == cfg.Global.Discord.Roles.Supporter &&
						role.ID != "" && cfg.Global.Discord.Roles.RoleCache.Supporter != role.ID {
						cfg.Global.Discord.Roles.RoleCache.Supporter = role.ID
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
			time.Sleep(2 * time.Second)

			if fact.FactIsRunning && fact.FactorioBooted && fact.NumPlayers == 0 {

				if fact.QueueReboot && !fact.DoUpdateFactorio {
					cwlog.DoLogCW("No players currently online, performing scheduled reboot.")
					fact.QuitFactorio("Server rebooting for maintenance.")
					break //We don't need to loop anymore, rebooting chat wire.

				} else if fact.QueueFactReboot && !fact.DoUpdateFactorio {
					cwlog.DoLogCW("Stopping Factorio for reboot.")
					fact.QuitFactorio("Rebooting factorio.")
					time.Sleep(time.Minute)

				} else if fact.DoUpdateFactorio {
					cwlog.DoLogCW("Stopping Factorio for update.")
					fact.QuitFactorio("Updating factorio.")
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
			time.Sleep(time.Second * 5)

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
						fact.FactChat(fact.AddFactColor("green", msg))
						fact.FactChat(fact.AddFactColor("black", msg))
						fact.FactChat(fact.AddFactColor("white", msg))
					}
					time.Sleep(2 * time.Minute)

					/* Reboot anyway */
					if glob.UpdateWarnCounter > glob.UpdateGraceMinutes {
						msg := "(SYSTEM) Rebooting for Factorio update!"
						fact.CMS(cfg.Local.Channel.ChatChannel, msg)
						glob.UpdateWarnCounter = 0
						fact.QuitFactorio("Rebooting for Factorio update: " + fact.NewVersion)
						break /* Stop looping */
					}
					glob.UpdateWarnCounter = (glob.UpdateWarnCounter + 1)
				} else {
					glob.UpdateWarnCounter = 0
					fact.QuitFactorio("Rebooting for Factorio update: " + fact.NewVersion)
					break /* Stop looping */
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
					if !fact.QueueReboot {
						fact.QueueReboot = true
						cwlog.DoLogCW("Reboot queued!")
					}
				} else if !failureReported {
					failureReported = true
					fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .queue file, ignoring.")
				}
			}
			/* Only if game is running */
			if fact.FactIsRunning && fact.FactorioBooted {
				/* Stop game */
				if _, err = os.Stat(".stop"); err == nil {
					if errb = os.Remove(".stop"); errb == nil {
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio stopping!")
						fact.FactAutoStart = false
						fact.QuitFactorio("Server stopping for maintenance.")
					} else if !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .stop file, ignoring.")
					}
				}

				/* Restart game */
				if _, err = os.Stat(".rfact"); err == nil {
					if errb = os.Remove(".rfact"); errb == nil {
						fact.QueueFactReboot = true
					} else if !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Failed to remove .rfact file, ignoring.")
					}
				}
			} else { /*  Only if game is NOT running */
				/* Start game */
				if _, err = os.Stat(".start"); err == nil {
					if errb = os.Remove(".start"); errb == nil {
						fact.FactAutoStart = true
						fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio starting!")
					} else if !failureReported {
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
			time.Sleep(time.Second * 5)

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

		time.Sleep(time.Second * 10)
		for glob.ServerRunning {
			if cfg.Local.Options.AutoUpdate {
				_, msg, err, upToDate := factUpdater.DoQuickLatest(false)
				if msg != "" {
					cwlog.DoLogCW(msg)
					if !err && !upToDate {
						myembed := embed.NewEmbed().
							SetTitle("Info").
							SetDescription(msg).
							SetColor(0xff0000).MessageEmbed
						disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, myembed)
					}
				}
				time.Sleep(time.Minute * 10)
				time.Sleep(time.Second * time.Duration(rand.Intn(300))) //Add 5 minutes of randomness
			} else {
				time.Sleep(time.Minute)
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

			time.Sleep(time.Second * 10)
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
						fact.CMS(cfg.Local.Channel.ChatChannel, msg)
						fact.FactChat(msg)
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
			time.Sleep(time.Minute * 15)
		}
	}()

	/****************************/
	/* Check for mod update     */
	/****************************/
	go func() {
		time.Sleep(time.Minute)

		for glob.ServerRunning &&
			cfg.Local.Options.ModUpdate {
			modupdate.CheckMods(false, false)

			time.Sleep(time.Hour * 3)
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
		var lastDur string
		for glob.ServerRunning {
			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted &&
				fact.NumPlayers > 0 {
				time.Sleep(time.Minute)

				fact.UpdateScheduleDesc()
				if fact.TillReset != "" && cfg.Local.Options.Schedule != "" {
					buf := "/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")"
					/* Don't write it, if nothing has changed */
					if !strings.EqualFold(buf, lastDur) {
						fact.WriteFact(buf)
					}

					lastDur = buf
				}
			}

			time.Sleep(time.Second)
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
