package support

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ChatWire/banlist"
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func LinuxSetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

//*******************
//Main threads/loops
//*******************

func MainLoops() {

	//Wait to start loops...
	time.Sleep(time.Second)

	go func() { //nested for 'reasons'

		//**************
		//Game watchdog
		//**************
		go func() {
			for glob.ServerRunning {
				time.Sleep(constants.WatchdogInterval)

				//Check for Factorio updates
				if !fact.IsFactRunning() && (fact.IsQueued() || fact.IsSetRebootBot() || fact.GetDoUpdateFactorio()) {
					if fact.GetDoUpdateFactorio() {
						fact.FactUpdate()
					}
					fact.DoExit(true)
				} else if fact.IsFactRunning() { //Currently running normally

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
				} else if !fact.IsFactRunning() && fact.IsSetAutoStart() && !fact.GetDoUpdateFactorio() { //Isn't running, but we should be
					//Dont relaunch if we are set to auto update

					if cfg.Global.PathData.ScriptInserterPath != "" {
						command := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.ScriptInserterPath
						out, errs := exec.Command(command, cfg.Local.ServerCallsign).Output()
						if errs != nil {
							botlog.DoLog(fmt.Sprintf("Unable to run soft-mod insert script. Details:\nout: %v\nerr: %v", string(out), errs))
						} else {
							botlog.DoLog("Soft-mod inserted into save file.")
						}
					}

					//Generate config file for Factorio server, if it fails stop everything.
					if !fact.GenerateFactorioConfig() {
						fact.SetAutoStart(false)
						fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to generate config file for Factorio server.")
						return
					}

					//Relaunch Throttling
					throt := fact.GetRelaunchThrottle()
					if throt > 0 {

						delay := throt * throt * 10

						if delay > 0 {
							botlog.DoLog(fmt.Sprintf("Automatically rebooting Factroio in %d seconds.", delay))
							for i := 0; i < delay*11 && throt > 0; i++ {
								time.Sleep(100 * time.Millisecond)
							}
						}
					}
					//Timer gets longer each reboot
					fact.SetRelaunchThrottle(throt + 1)

					//Prevent us from distrupting updates
					fact.FactorioLaunchLock.Lock()

					var err error
					var tempargs []string

					//Factorio launch parameters
					rconport := cfg.Local.Port + cfg.Global.RconPortOffset
					rconportStr := fmt.Sprintf("%v", rconport)
					rconpass := cfg.Global.RconPass
					port := cfg.Local.Port
					postStr := fmt.Sprintf("%v", port)
					serversettings := cfg.Global.PathData.FactorioServersRoot +
						cfg.Global.PathData.FactorioHomePrefix +
						cfg.Local.ServerCallsign + "/" +
						constants.ServSettingsName

					tempargs = append(tempargs, "--start-server-load-latest")
					tempargs = append(tempargs, "--rcon-port")
					tempargs = append(tempargs, rconportStr)

					tempargs = append(tempargs, "--rcon-password")
					tempargs = append(tempargs, rconpass)

					tempargs = append(tempargs, "--port")
					tempargs = append(tempargs, postStr)

					tempargs = append(tempargs, "--server-settings")
					tempargs = append(tempargs, serversettings)

					//Auth Server Bans ( world bans )
					if cfg.Global.AuthServerBans {
						tempargs = append(tempargs, "--use-authserver-bans")
					}

					//Whitelist
					if cfg.Local.DoWhitelist {
						tempargs = append(tempargs, "--use-server-whitelist")
						tempargs = append(tempargs, "true")
					}

					//Write or delete whitelist
					count := fact.WriteWhitelist()
					if count > 0 && cfg.Local.DoWhitelist {
						botlog.DoLog(fmt.Sprintf("Whitelist of %v players written.", count))
					}

					//Run Factorio
					var cmd *exec.Cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)

					/*Hide RCON password and port*/
					for i, targ := range tempargs {
						if targ == rconpass {
							tempargs[i] = "***private***"
						} else if targ == rconportStr {
							//funny, and impossible port number
							tempargs[i] = "69420"
						}
					}
					botlog.DoLog("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))

					LinuxSetProcessGroup(cmd)
					//Connect Factorio stdout to a buffer for processing
					fact.GameBuffer = new(bytes.Buffer)
					logwriter := io.MultiWriter(fact.GameBuffer)
					cmd.Stdout = logwriter
					//Stdin
					tpipe, errp := cmd.StdinPipe()

					//Factorio is not happy.
					if errp != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
						//close lock
						fact.FactorioLaunchLock.Unlock()
						fact.DoExit(true)
						return
					}

					//Save pipe
					if tpipe != nil && err == nil {
						fact.PipeLock.Lock()
						fact.Pipe = tpipe
						fact.PipeLock.Unlock()
					}

					//Handle launch errors
					err = cmd.Start()
					if err != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
						//close lock
						fact.FactorioLaunchLock.Unlock()
						fact.DoExit(true)
						return
					}

					//Okay, factorio is running now, prep
					fact.SetModLoadString(constants.Unknown)
					fact.ModLoadString = constants.Unknown //Reset loaded mod list
					fact.SetFactRunning(true, false)
					fact.SetFactorioBooted(false)

					fact.SetGameTime(constants.Unknown)
					fact.SetNoResponseCount(0)
					botlog.DoLog("Factorio booting...")

					//close lock
					fact.FactorioLaunchLock.Unlock()
				}
			}
		}()

		//*******************************
		//Discord stats update
		//*******************************
		go func() {
			time.Sleep(5 * time.Minute)
			for glob.ServerRunning {
				time.Sleep(5 * time.Second)
				if cfg.Local.WriteStatsDisc {

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

					totalstat := fmt.Sprintf("total-%v", (numtrust + numregulars + numadmin + numother))
					memberstat := fmt.Sprintf("members-%v", numtrust)
					regularstat := fmt.Sprintf("regulars-%v", numregulars)

					if glob.LastTotalStat != totalstat && cfg.Global.DiscordData.StatTotalChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatTotalChannelID, &discordgo.ChannelEdit{Name: totalstat, Position: 1})
						glob.LastTotalStat = totalstat
						if err != nil {
							botlog.DoLog(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					if glob.LastMemberStat != memberstat && cfg.Global.DiscordData.StatMemberChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatMemberChannelID, &discordgo.ChannelEdit{Name: memberstat, Position: 2})
						glob.LastMemberStat = memberstat
						if err != nil {
							botlog.DoLog(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					if glob.LastRegularStat != regularstat && cfg.Global.DiscordData.StatRegularsChannelID != "" {
						_, err := disc.DS.ChannelEditComplex(cfg.Global.DiscordData.StatRegularsChannelID, &discordgo.ChannelEdit{Name: regularstat, Position: 3})
						glob.LastRegularStat = regularstat
						if err != nil {
							botlog.DoLog(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

				}

			}
		}()

		//*******************************
		//Look for lockers
		//*******************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(time.Second)

				fact.LockerLock.Lock()

				if fact.LockerStart {
					if time.Since(fact.LockerDetectStart) > time.Second*6 {
						fact.LockerStart = false
						fact.LockerDetectStart = time.Now()

						go func() {
							botlog.DoLog("Possible locker")
							fact.WriteFact("/chat Locker detected, rebooting.")
							fact.CMS(cfg.Local.ChannelData.ChatID, "Possible locker detected, rebooting.")

							time.Sleep(time.Second)
							//fact.QuitFactorio()
							fact.SetRelaunchThrottle(0)
							fact.SetNoResponseCount(0)
							fact.WriteFact("/quit")
						}()
					}
				}
				fact.LockerLock.Unlock()
			}
		}()

		//*******************************
		//Watch ban file
		//*******************************
		go banlist.WatchBanFile()

		//*******************************
		//Send buffered messages to Discord, batched.
		//*******************************
		go func() {
			for glob.ServerRunning {

				if disc.DS != nil {

					//Check if buffer is active
					active := false
					disc.CMSBufferLock.Lock()
					if disc.CMSBuffer != nil {
						active = true
					}
					disc.CMSBufferLock.Unlock()

					//If buffer is active, sleep and wait for it to fill up
					if active {
						time.Sleep(constants.CMSRate)

						//Waited for buffer to fill up, grab and clear buffers
						disc.CMSBufferLock.Lock()
						lcopy := disc.CMSBuffer
						disc.CMSBuffer = nil
						disc.CMSBufferLock.Unlock()

						if lcopy != nil {

							var factmsg []string
							var moder []string

							for _, msg := range lcopy {
								if msg.Channel == cfg.Local.ChannelData.ChatID {
									factmsg = append(factmsg, msg.Text)
								} else if msg.Channel == cfg.Global.DiscordData.ReportChannelID {
									moder = append(moder, msg.Text)
								} else {
									disc.SmartWriteDiscord(msg.Channel, msg.Text)
								}
							}

							//Send out buffer, split up if needed
							//Factorio
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

							//Moderation
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

						//Don't send any more messages for a while (throttle)
						time.Sleep(constants.CMSRestTime)
					}

				}

				//Sleep for a moment before checking buffer again
				time.Sleep(constants.CMSPollRate)
			}
		}()

		//**********************
		//Check players online
		//**********************
		//Safety, in case player count gets off
		//Also helps detect servers crash/dead while paused
		go func() {
			for glob.ServerRunning {
				time.Sleep(5 * time.Minute)

				if fact.IsFactRunning() {
					fact.WriteFact("/p o c")
				}
			}
		}()

		//**********************************
		//Delete expired registration codes
		//**********************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(30 * time.Second)

				t := time.Now()

				glob.PasswordListLock.Lock()
				for _, pass := range glob.PassList {
					if (t.Unix() - pass.Time) > 300 {
						botlog.DoLog("Invalidating old unused access code for user: " + disc.GetNameFromID(pass.DiscID, false))
						delete(glob.PassList, pass.DiscID)
					}
				}
				glob.PasswordListLock.Unlock()
			}
		}()

		//***************************************
		//Save vote-rewind data async
		//***************************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(10 * time.Second)

				glob.VoteBoxLock.Lock()

				//Save if dirty
				if glob.VoteBox.Dirty {
					fact.WriteRewindVotes()
					glob.VoteBox.Dirty = false
				}
				glob.VoteBoxLock.Unlock()
			}
		}()

		//****************************************************
		//Slow-connect, helps players catch up on large maps
		//****************************************************
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
		//*******************************
		//Save database, if marked dirty
		//*******************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(1 * time.Second)

				wasDirty := false

				glob.PlayerListDirtyLock.Lock()

				if glob.PlayerListDirty {
					glob.PlayerListDirty = false
					wasDirty = true
					//Prevent recursive lock
					go func() {
						time.Sleep(3 * time.Second)
						botlog.DoLog("Database marked dirty, saving.")
						fact.WritePlayers()
					}()
				}
				glob.PlayerListDirtyLock.Unlock()

				//Sleep after saving
				if wasDirty {
					time.Sleep(10 * time.Second)
				}
			}
		}()

		//**********************************************************
		//Save database (less often), if last seen is marked dirty
		//**********************************************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(5 * time.Minute)
				glob.PlayerListSeenDirtyLock.Lock()

				if glob.PlayerListSeenDirty {
					glob.PlayerListSeenDirty = false

					//Prevent recursive lock
					go func() {
						//botlog.DoLog("Database last seen flagged, saving.")
						fact.WritePlayers()
					}()
				}
				glob.PlayerListSeenDirtyLock.Unlock()
			}
		}()

		//***********************************
		//Database file modification watching
		//***********************************
		go fact.WatchDatabaseFile()

		//Read database, if the file was modifed
		go func() {
			updated := false

			for glob.ServerRunning {

				time.Sleep(1 * time.Second)

				//Detect update
				glob.PlayerListUpdatedLock.Lock()
				if glob.PlayerListUpdated {
					updated = true
					glob.PlayerListUpdated = false
				}
				glob.PlayerListUpdatedLock.Unlock()

				if updated {
					updated = false

					//botlog.DoLog("Database file modified, loading.")
					fact.LoadPlayers()

					//Sleep after reading
					time.Sleep(5 * time.Second)
				}

			}
		}()

		//**************************
		//Get Guild information
		//Needed for Discord roles
		//**************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(5 * time.Second)

				disc.GuildLock.Lock()

				//Get guild id, if we need it

				if disc.Guild == nil && disc.DS != nil {
					var nguild *discordgo.Guild
					var err error

					// Attempt to get the guild from the state,
					// If there is an error, fall back to the restapi.
					nguild, err = disc.DS.State.Guild(cfg.Global.DiscordData.GuildID)
					if err != nil {
						nguild, err = disc.DS.Guild(cfg.Global.DiscordData.GuildID)
						if err != nil {
							botlog.DoLog("Failed to get valid guild data, giving up.")
							disc.GuildLock.Unlock()
							break
						}
					}

					if err != nil {
						botlog.DoLog(fmt.Sprintf("Was unable to get guild data from GuildID: %s", err))
						disc.GuildLock.Unlock()
						break
					}
					if nguild == nil || err != nil {
						disc.Guildname = constants.Unknown
						botlog.DoLog("Guild data came back nil.")
						disc.GuildLock.Unlock()
						break
					} else {

						//Guild found, exit loop
						disc.Guild = nguild
						disc.Guildname = nguild.Name
						botlog.DoLog("Guild data linked.")
					}
				}

				//Update role IDs
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
						botlog.DoLog("Role IDs updated.")
						cfg.WriteGCfg()
					}
				}
				disc.GuildLock.Unlock()

				time.Sleep(time.Minute * 5)
			}
		}()

		//**************************
		//Update patreon/nitro players
		//**************************
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

					//Live update server description
					if disc.RoleListUpdated {
						//goroutine, avoid deadlock
						go fact.GenerateFactorioConfig()
					}
					disc.RoleListUpdated = false
					disc.RoleListLock.Unlock()
				}
			}
		}()

		//************************************
		//Reboot if queued, when server empty
		//************************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(1 * time.Second)

				if fact.IsQueued() && fact.GetNumPlayers() == 0 && !fact.GetDoUpdateFactorio() {
					if fact.IsFactRunning() {
						botlog.DoLog("No players currently online, performing scheduled reboot.")
						fact.QuitFactorio()
						break //We don't need to loop anymore
					}
				}
			}
		}()

		//*******************************************
		//Bug players if there is an pending update
		//*******************************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(5 * time.Second)

				if cfg.Local.AutoUpdate {
					if fact.IsFactRunning() && fact.NewVersion != constants.Unknown {
						if fact.GetNumPlayers() > 0 {

							numwarn := fact.GetUpdateWarnCounter()

							//Warn users
							if numwarn < glob.UpdateGraceMinutes {
								msg := fmt.Sprintf("(SYSTEM) Factorio update waiting (%v), please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!", fact.NewVersion, glob.UpdateGraceMinutes-numwarn)
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat " + fact.AddFactColor("red", msg))
							}
							time.Sleep(1 * time.Minute)

							//Reboot anyway
							if numwarn > glob.UpdateGraceMinutes {
								msg := "(SYSTEM) Rebooting for Factorio update."
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat " + fact.AddFactColor("red", msg))
								fact.SetUpdateWarnCounter(0)
								fact.QuitFactorio()
								break //Stop looping
							}
							fact.SetUpdateWarnCounter(numwarn + 1)
						} else {
							fact.SetUpdateWarnCounter(0)
							fact.QuitFactorio()
							break //Stop looping
						}
					}
				}
			}
		}()

		//*******************
		//Check signal files
		//*******************
		go func() {
			clearOldSignals()
			failureReported := false
			for glob.ServerRunning {

				time.Sleep(5 * time.Second)

				var err error
				var errb error

				//Queued reboots, regardless of game state
				if _, err = os.Stat(".queue"); err == nil {
					if errb = os.Remove(".queue"); errb == nil {
						if !fact.IsQueued() {
							fact.SetQueued(true)
							botlog.DoLog("Reboot queued!")
						}
					} else if errb != nil && !failureReported {
						failureReported = true
						fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .queue file, ignoring.")
					}
				}
				//Halt, regardless of game state
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

				//Only if game is running
				if fact.IsFactRunning() {
					//Quick reboot
					//This should eventually grab save name from file
					if _, err = os.Stat(".qrestart"); err == nil {
						if errb = os.Remove(".qrestart"); errb == nil {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio quick restarting!")
							fact.QuitFactorio()
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .qrestart file, ignoring.")
						}
					}
					//Stop game
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
					//New map
					if _, err = os.Stat(".newmap"); err == nil {
						if errb = os.Remove(".newmap"); errb == nil {
							fact.Map_reset("")
						} else if errb != nil && !failureReported {
							failureReported = true
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Failed to remove .stop file, ignoring.")
						}
					}
					//Message
					if _, err = os.Stat(".message"); err == nil {
						data, errc := os.ReadFile(".message")
						if errb = os.Remove(".message"); errb == nil {
							if errc == nil && data != nil {
								message := string(data)
								msglen := len(message)
								if msglen > 5 && msglen < 250 {
									message = strings.ReplaceAll(message, "\n", "") //replace newline
									message = strings.ReplaceAll(message, "\r", "") //replace return
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
				} else { //Only if game is NOT running
					//Start game
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

		//*********************************
		//Fix lost connection to log files
		//*********************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(time.Second * 15)

				var err error
				if _, err = os.Stat(glob.BotLogName); err != nil {

					glob.BotLogDesc.Close()
					glob.BotLogDesc = nil
					botlog.StartBotLog()
					botlog.DoLog("BotLog file was deleted, recreated.")
				}

				if _, err = os.Stat(glob.GameLogName); err != nil {
					glob.GameLogDesc.Close()
					glob.GameLogDesc = nil
					botlog.StartGameLog()
					botlog.DoLogGame("GameLog file was deleted, recreated.")
				}
			}
		}()

		//****************************
		// Check for factorio updates
		//****************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(60 * time.Minute)
				fact.CheckFactUpdate(false)

			}
		}()

		//****************************
		// Refresh channel names
		//****************************
		go func() {

			for glob.ServerRunning {
				fact.UpdateChannelName()

				disc.UpdateChannelLock.Lock()
				chname := disc.NewChanName
				oldchname := disc.OldChanName
				disc.UpdateChannelLock.Unlock()

				if oldchname != chname {
					fact.DoUpdateChannelName()
					time.Sleep(time.Minute * 1)
				} else {

					time.Sleep(5 * time.Second)
				}
			}
		}()

		//****************************
		// Capture man-minutes
		//****************************
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

	//After starting loops, wait here for process signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	_ = os.Remove("cw.lock")
	fact.SetAutoStart(false)
	fact.SetBotReboot(false)
	fact.SetQueued(false)
	fact.QuitFactorio()
	for x := 0; x < constants.MaxFactorioCloseWait && fact.IsFactRunning(); x++ {
		time.Sleep(time.Second)
	}
	fact.DoExit(false)
}

//Delete old signal files
func clearOldSignals() {
	if err := os.Remove(".qrestart"); err == nil {
		botlog.DoLog("old .qrestart removed.")
	}
	if err := os.Remove(".queue"); err == nil {
		botlog.DoLog("old .queue removed.")
	}
	if err := os.Remove(".stop"); err == nil {
		botlog.DoLog("old .stop removed.")
	}
	if err := os.Remove(".newmap"); err == nil {
		botlog.DoLog("old .newmap removed.")
	}
	if err := os.Remove(".message"); err == nil {
		botlog.DoLog("old .message removed.")
	}
	if err := os.Remove(".start"); err == nil {
		botlog.DoLog("old .start removed.")
	}
	if err := os.Remove(".halt"); err == nil {
		botlog.DoLog("old .halt removed.")
	}
}
