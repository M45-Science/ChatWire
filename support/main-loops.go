package support

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

/********************
 * Main threads/loops
 ********************/

func MainLoops() {

	go func() { /* nested for 'reasons' */

		/***************
		 * Game watchdog
		 ***************/
		go func() {
			for glob.ServerRunning {
				time.Sleep(constants.WatchdogInterval)

				/* Check for updates */
				if !fact.IsFactRunning() && (fact.IsQueued() || fact.IsSetRebootCW() || fact.GetDoUpdateFactorio()) {
					if fact.GetDoUpdateFactorio() {
						fact.FactUpdate()
					}
					fact.DoExit(true)

					/* We are running normally */
				} else if fact.IsFactRunning() {

					nores := 0
					if fact.GetPausedTicks() <= constants.PauseThresh {
						glob.NoResponseCountLock.Lock()
						glob.NoResponseCount = glob.NoResponseCount + 1
						nores = glob.NoResponseCount
						glob.NoResponseCountLock.Unlock()

						fact.WriteFact("/time")
					}
					if nores == 120 {
						fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio unresponsive for over two minutes... rebooting.")
						fact.SetRelaunchThrottle(0)
						fact.QuitFactorio()
						for x := 0; x < constants.MaxFactorioCloseWait && fact.IsFactRunning(); x++ {
							time.Sleep(time.Second)
						}
					}

					/* We aren't running, but should be! */
				} else if !fact.IsFactRunning() && fact.IsSetAutoStart() && !fact.GetDoUpdateFactorio() {
					/* Dont relaunch if we are set to auto update */

					launchFactortio()
				}
			}
		}()

		/********************************
		 * Discord stats update
		 ********************************/
		go func() {
			time.Sleep(5 * time.Minute)
			for glob.ServerRunning {
				time.Sleep(5 * time.Second)
				if cfg.Local.WriteStatsDisc {

					banlist.BanListLock.Lock()
					banCount := len(banlist.BanList)
					banlist.BanListLock.Unlock()

					numreg := 0
					numnew := 0
					numtrust := 0
					numregulars := 0
					numadmin := 0
					numother := 0

					glob.PlayerListLock.RLock()
					for _, player := range glob.PlayerList {
						if player.ID != "" {
							numreg++
						}

						if player.Level == 0 {
							numnew++
						} else if player.Level == 1 {
							numtrust++
						} else if player.Level == 2 {
							numregulars++
						} else if player.Level == 255 {
							numadmin++
						} else {
							numother++
						}
					}
					glob.PlayerListLock.RUnlock()

					totalstat := fmt.Sprintf("total-%v", (numnew + numtrust + numregulars + numadmin + banCount))
					memberstat := fmt.Sprintf("members-%v", numtrust)
					regularstat := fmt.Sprintf("regulars-%v", numregulars)

					if glob.LastRegularStat != regularstat && cfg.Global.DiscordData.StatRegularsChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatRegularsChannelID, &discordgo.ChannelEdit{Name: regularstat, Position: 1})
						glob.LastRegularStat = regularstat
						if err != nil {
							cwlog.DoLogCW(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					if glob.LastMemberStat != memberstat && cfg.Global.DiscordData.StatMemberChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatMemberChannelID, &discordgo.ChannelEdit{Name: memberstat, Position: 2})
						glob.LastMemberStat = memberstat
						if err != nil {
							cwlog.DoLogCW(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					banstat := fmt.Sprintf("bans-%v", banCount)
					if glob.LastBanStat != banstat {
						if cfg.Global.DiscordData.StatBanChannelID != "" {
							_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatBanChannelID, &discordgo.ChannelEdit{Name: banstat, Position: 3})
							glob.LastBanStat = banstat
							if err != nil {
								cwlog.DoLogCW(err.Error())
							}
							time.Sleep(5 * time.Minute)
						}
					}

					if glob.LastTotalStat != totalstat && cfg.Global.DiscordData.StatTotalChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatTotalChannelID, &discordgo.ChannelEdit{Name: totalstat, Position: 4})
						glob.LastTotalStat = totalstat
						if err != nil {
							cwlog.DoLogCW(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

				}

			}
		}()

		/********************************
		 * Look for lockers
		 ********************************/
		go func() {
			for glob.ServerRunning {
				time.Sleep(time.Second)

				fact.LockerLock.Lock()

				if fact.LockerStart {
					if time.Since(fact.LockerDetectStart) > time.Second*10 && fact.LastLockerName != "" {
						fact.LockerDetectStart = time.Now()
						fact.LockerStart = false

						msg := "Locker bug detected (" + fact.LastLockerName + "), kicking."
						cwlog.DoLogCW(msg)
						fact.WriteFact("/chat " + msg)
						fact.CMS(cfg.Local.ChannelData.ChatID, msg)
						fact.WriteFact("/kick " + fact.LastLockerName)
					}
				}

				fact.LockerLock.Unlock()
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
								if msg.Channel == cfg.Local.ChannelData.ChatID {
									factmsg = append(factmsg, msg.Text)
								} else if msg.Channel == cfg.Global.DiscordData.ReportChannelID {
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
									disc.SmartWriteDiscord(cfg.Local.ChannelData.ChatID, buf)
									buf = line
								} else {
									buf = buf + "\n" + line
								}
							}
							if buf != "" {
								disc.SmartWriteDiscord(cfg.Local.ChannelData.ChatID, buf)
							}

							/* Moderation */
							buf = ""
							for _, line := range moder {
								oldlen := len(buf) + 1
								addlen := len(line)
								if oldlen+addlen >= 2000 {
									disc.SmartWriteDiscord(cfg.Global.DiscordData.ReportChannelID, buf)
									buf = line
								} else {
									buf = buf + "\n" + line
								}
							}
							if buf != "" {
								disc.SmartWriteDiscord(cfg.Global.DiscordData.ReportChannelID, buf)
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

		/***********************
		 * Check players online
		 ***********************/
		/* Safety, in case player count gets off
		 * Also helps detect servers crash/dead while paused */
		go func() {
			for glob.ServerRunning {
				time.Sleep(1 * time.Minute)

				if fact.IsFactRunning() {
					fact.WriteFact("/p o c")
				}
			}
		}()

		/***********************************
		 * Delete expired registration codes
		 ***********************************/
		go func() {

			for glob.ServerRunning {
				time.Sleep(10 * time.Second)

				t := time.Now()

				glob.PasswordListLock.Lock()
				for _, pass := range glob.PassList {
					if (t.Unix() - pass.Time) > 300 {
						cwlog.DoLogCW("Invalidating old unused access code for player: " + disc.GetNameFromID(pass.DiscID, false))
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
				time.Sleep(10 * time.Second)

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

				if cfg.Local.SlowConnect.SlowConnect {

					fact.ConnectPauseLock.Lock()

					if fact.ConnectPauseTimer > 0 {
						if tn.Unix()-fact.ConnectPauseTimer >= 120 {
							fact.ConnectPauseTimer = 0
							fact.ConnectPauseCount = 0

							buf := "Catch-up taking over two minutes, returning to normal speed."
							fact.CMS(cfg.Local.ChannelData.ChatID, buf)
							fact.WriteFact("/chat (SYSTEM) " + buf)

							if cfg.Local.SlowConnect.DefaultSpeed > 0.0 {
								fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.SlowConnect.DefaultSpeed))
							} else {
								fact.WriteFact("/gspeed 1.0")
							}
						}
					}

					fact.ConnectPauseLock.Unlock()

				}
			}
		}()
		/********************************
		 * Save database, if marked dirty
		 ********************************/
		go func() {
			for glob.ServerRunning {
				time.Sleep(1 * time.Second)

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

				time.Sleep(1 * time.Second)

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
				time.Sleep(1 * time.Second)

				disc.GuildLock.Lock()

				/* Get guild id, if we need it */

				if disc.Guild == nil && disc.DS != nil {
					var nguild *discordgo.Guild
					var err error

					/*  Attempt to get the guild from the state,
					 *  If there is an error, fall back to the restapi. */
					nguild, err = disc.DS.State.Guild(cfg.Global.DiscordData.GuildID)
					if err != nil {
						nguild, err = disc.DS.Guild(cfg.Global.DiscordData.GuildID)
						if err != nil {
							cwlog.DoLogCW("Failed to get valid guild data, giving up.")
							disc.GuildLock.Unlock()
							break
						}
					}

					if err != nil {
						cwlog.DoLogCW(fmt.Sprintf("Was unable to get guild data from GuildID: %s", err))
						disc.GuildLock.Unlock()
						break
					}
					if nguild == nil || err != nil {
						disc.Guildname = constants.Unknown
						cwlog.DoLogCW("Guild data came back nil.")
						disc.GuildLock.Unlock()
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
						if cfg.Global.RoleData.ModeratorRoleName != "" &&
							role.Name == cfg.Global.RoleData.ModeratorRoleName &&
							role.ID != "" && cfg.Global.RoleData.ModeratorRoleID != role.ID {
							cfg.Global.RoleData.ModeratorRoleID = role.ID
							changed = true

						} else if cfg.Global.RoleData.RegularRoleName != "" &&
							role.Name == cfg.Global.RoleData.RegularRoleName &&
							role.ID != "" && cfg.Global.RoleData.RegularRoleID != role.ID {
							cfg.Global.RoleData.RegularRoleID = role.ID
							changed = true

						} else if cfg.Global.RoleData.MemberRoleName != "" &&
							role.Name == cfg.Global.RoleData.MemberRoleName &&
							role.ID != "" && cfg.Global.RoleData.MemberRoleID != role.ID {
							cfg.Global.RoleData.MemberRoleID = role.ID
							changed = true

						} else if cfg.Global.RoleData.NewRoleName != "" &&
							role.Name == cfg.Global.RoleData.NewRoleName &&
							role.ID != "" && cfg.Global.RoleData.NewRoleID != role.ID {
							cfg.Global.RoleData.NewRoleID = role.ID
							changed = true
						} else if cfg.Global.RoleData.PatreonRoleName != "" &&
							role.Name == cfg.Global.RoleData.PatreonRoleName &&
							role.ID != "" && cfg.Global.RoleData.PatreonRoleID != role.ID {
							cfg.Global.RoleData.PatreonRoleID = role.ID
							changed = true
						} else if cfg.Global.RoleData.NitroRoleName != "" &&
							role.Name == cfg.Global.RoleData.NitroRoleName &&
							role.ID != "" && cfg.Global.RoleData.NitroRoleID != role.ID {
							cfg.Global.RoleData.NitroRoleID = role.ID
							changed = true
						}
					}
					if changed {
						cwlog.DoLogCW("Role IDs updated.")
						cfg.WriteGCfg()
					}
				}
				disc.GuildLock.Unlock()

				time.Sleep(time.Second * 30)
			}
		}()

		/*******************************
		 * Update patreon/nitro players
		 *******************************/
		go func() {
			for glob.ServerRunning {
				time.Sleep(time.Minute)
				if fact.IsFactorioBooted() {
					disc.UpdateRoleList()

					disc.RoleListLock.Lock()
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
					disc.RoleListLock.Unlock()
				}
			}
		}()

		/************************************
		 * Reboot if queued, when server empty
		 ************************************/
		go func() {

			for glob.ServerRunning {
				time.Sleep(2 * time.Second)

				if fact.IsQueued() && fact.GetNumPlayers() == 0 && !fact.GetDoUpdateFactorio() {
					if fact.IsFactRunning() {
						cwlog.DoLogCW("No players currently online, performing scheduled reboot.")
						fact.QuitFactorio()
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
				time.Sleep(5 * time.Second)

				if cfg.Local.AutoUpdate {
					if fact.IsFactRunning() && fact.NewVersion != constants.Unknown {
						if fact.GetNumPlayers() > 0 {

							numwarn := fact.GetUpdateWarnCounter()

							/* Warn players */
							if numwarn < glob.UpdateGraceMinutes {
								msg := fmt.Sprintf("(SYSTEM) Factorio update waiting (%v), please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!", fact.NewVersion, glob.UpdateGraceMinutes-numwarn)
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat " + fact.AddFactColor("red", msg))
							}
							time.Sleep(1 * time.Minute)

							/* Reboot anyway */
							if numwarn > glob.UpdateGraceMinutes {
								msg := "(SYSTEM) Rebooting for Factorio update."
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat " + fact.AddFactColor("red", msg))
								fact.SetUpdateWarnCounter(0)
								fact.QuitFactorio()
								break /* Stop looping */
							}
							fact.SetUpdateWarnCounter(numwarn + 1)
						} else {
							fact.SetUpdateWarnCounter(0)
							fact.QuitFactorio()
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
						if !fact.IsQueued() {
							fact.SetQueued(true)
							cwlog.DoLogCW("Reboot queued!")
						}
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .queue file, ignoring.")
					}
				}
				/* Halt, regardless of game state */
				if _, err = os.Stat(".halt"); err == nil {
					if errb = os.Remove(".halt"); errb == nil {
						if fact.IsFactRunning() {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "ChatWire is halting, closing Factorio.")
							fact.SetAutoStart(false)
							fact.QuitFactorio()
							for x := 0; x < constants.MaxFactorioCloseWait && fact.IsFactRunning(); x++ {
								time.Sleep(time.Second)
							}
							fact.DoExit(false)
						} else {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "ChatWire is halting.")
							fact.DoExit(false)
						}
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .halt file, ignoring.")
					}
				}

				/* Only if game is running */
				if fact.IsFactRunning() {
					/* Quick reboot
					 * This should eventually grab save name from file */
					if _, err = os.Stat(".qrestart"); err == nil {
						if errb = os.Remove(".qrestart"); errb == nil {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio quick restarting!")
							fact.QuitFactorio()
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .qrestart file, ignoring.")
						}
					}
					/* Stop game */
					if _, err = os.Stat(".stop"); err == nil {
						if errb = os.Remove(".stop"); errb == nil {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio stopping!")
							fact.SetAutoStart(false)
							fact.QuitFactorio()
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .stop file, ignoring.")
						}
					}
					/* New map */
					if _, err = os.Stat(".newmap"); err == nil {
						if errb = os.Remove(".newmap"); errb == nil {
							fact.Map_reset("")
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .stop file, ignoring.")
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
									fact.Map_reset(message)
								} else {
									fact.LogCMS(cfg.Local.ChannelData.ChatID, ".message text is invalid, ignoring.")
								}
							}
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .message file, ignoring.")
						}
					}
				} else { /*  Only if game is NOT running */
					/* Start game */
					if _, err = os.Stat(".start"); err == nil {
						if errb = os.Remove(".start"); errb == nil {
							fact.SetAutoStart(true)
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio starting!")
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .start file, ignoring.")
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
				time.Sleep(time.Second * 15)

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
				time.Sleep(time.Hour)
				fact.CheckFactUpdate(false)

			}
		}()

		/****************************
		* Refresh channel names
		****************************/
		go func() {

			for glob.ServerRunning {
				fact.UpdateChannelName()

				disc.UpdateChannelLock.Lock()
				chname := disc.NewChanName
				oldchname := disc.OldChanName
				disc.UpdateChannelLock.Unlock()

				if oldchname != chname {
					fact.DoUpdateChannelName()
					time.Sleep(time.Second * 30)
				} else {

					time.Sleep(5 * time.Second)
				}
			}
		}()

		/****************************
		 * Capture man-minutes
		 ****************************/
		go func() {
			for glob.ServerRunning {
				time.Sleep(time.Minute)
				nump := fact.GetNumPlayers()

				fact.ManMinutesLock.Lock()
				if nump > 0 {
					fact.ManMinutes = (fact.ManMinutes + nump)
				}
				fact.ManMinutesLock.Unlock()
			}
		}()
	}()

	/* After starting loops, wait here for process signals */
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	_ = os.Remove("cw.lock")
	fact.SetAutoStart(false)
	fact.SetCWReboot(false)
	fact.SetQueued(false)
	fact.QuitFactorio()
	for x := 0; x < constants.MaxFactorioCloseWait && fact.IsFactRunning(); x++ {
		time.Sleep(time.Second)
	}
	fact.DoExit(false)
}
