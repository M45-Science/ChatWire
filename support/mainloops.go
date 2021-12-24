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

	go func() {

		//**************
		//Game watchdog
		//**************
		go func() {
			for glob.ServerRunning {
				time.Sleep(constants.WatchdogInterval)

				if !fact.IsFactRunning() && (fact.IsQueued() || fact.IsSetRebootBot() || fact.GetDoUpdateFactorio()) {
					if fact.GetDoUpdateFactorio() {
						fact.FactUpdate()
					}
					fact.DoExit()
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
						for x := 0; x < 60 && fact.IsFactRunning(); x++ {
							time.Sleep(time.Second)
						}
					}
				} else if !fact.IsFactRunning() && fact.IsSetAutoStart() && !fact.GetDoUpdateFactorio() { //Isn't running, but we should be
					//Dont relaunch if we are set to auto update

					command := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.ScriptInserterPath
					out, errs := exec.Command(command, cfg.Local.ServerCallsign).Output()
					if errs != nil {
						botlog.DoLog(fmt.Sprintf("Unable to run soft-mod insert script. Details:\nout: %v\nerr: %v", string(out), errs))
					} else {
						botlog.DoLog("Soft-mod inserted into save file.")
					}

					time.Sleep(2 * time.Second)

					//Relaunch Throttle
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

					fact.SetRelaunchThrottle(throt + 1)

					//Prevent us from distrupting updates
					glob.FactorioLaunchLock.Lock()

					var err error
					var tempargs []string

					rconport := cfg.Local.Port + cfg.Global.RconPortOffset
					rconportStr := fmt.Sprintf("%v", rconport)
					rconpass := cfg.Global.RconPass
					port := cfg.Local.Port
					postStr := fmt.Sprintf("%v", port)
					serversettings := cfg.Global.PathData.FactorioServersRoot +
						cfg.Global.PathData.FactorioHomePrefix +
						cfg.Local.ServerCallsign + "/" +
						"server-settings.json"

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
					if cfg.Local.SoftModOptions.DoWhitelist {
						tempargs = append(tempargs, "--use-server-whitelist")
						tempargs = append(tempargs, "true")
					}

					//Write or delete whitelist
					count := fact.WriteWhitelist()
					if count > 0 && cfg.Local.SoftModOptions.DoWhitelist {
						fact.LogCMS(cfg.Local.ChannelData.ChatID, (fmt.Sprintf("Whitelist of %v players written.", count)))
					}

					var cmd *exec.Cmd
					cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)

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
					glob.GameBuffer = new(bytes.Buffer)
					logwriter := io.MultiWriter(glob.GameLogDesc, glob.GameBuffer)
					cmd.Stdout = logwriter

					tpipe, errp := cmd.StdinPipe()

					if errp != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
						//close lock
						glob.FactorioLaunchLock.Unlock()
						fact.DoExit()
						return
					}

					if tpipe != nil && err == nil {
						glob.PipeLock.Lock()
						glob.Pipe = tpipe
						glob.PipeLock.Unlock()
					}

					//Pre-launch prep
					fact.SetFactRunning(true, false)
					fact.SetFactorioBooted(false)

					fact.SetGameTime(constants.Unknown)
					fact.SetSaveTimer()
					fact.SetNoResponseCount(0)

					err = cmd.Start()

					if err != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
						//close lock
						glob.FactorioLaunchLock.Unlock()
						fact.DoExit()
						return
					}
					botlog.DoLog("Factorio booting...")

					//close lock
					glob.FactorioLaunchLock.Unlock()
				}
			}
		}()

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
						_, err := glob.DS.ChannelEditComplex(cfg.Global.DiscordData.StatTotalChannelID, &discordgo.ChannelEdit{Name: totalstat, Position: 1})
						glob.LastTotalStat = totalstat
						if err != nil {
							botlog.DoLog(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					if glob.LastMemberStat != memberstat && cfg.Global.DiscordData.StatMemberChannelID != "" {
						_, err := glob.DS.ChannelEditComplex(cfg.Global.DiscordData.StatMemberChannelID, &discordgo.ChannelEdit{Name: memberstat, Position: 2})
						glob.LastMemberStat = memberstat
						if err != nil {
							botlog.DoLog(err.Error())
						}
						time.Sleep(5 * time.Minute)
					}

					if glob.LastRegularStat != regularstat && cfg.Global.DiscordData.StatRegularsChannelID != "" {
						_, err := glob.DS.ChannelEditComplex(cfg.Global.DiscordData.StatRegularsChannelID, &discordgo.ChannelEdit{Name: regularstat, Position: 3})
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
		//CMS Output from buffer, batched
		//*******************************
		go func() {
			for glob.ServerRunning {

				if glob.DS != nil {

					//Check if buffer is active
					active := false
					glob.CMSBufferLock.Lock()
					if glob.CMSBuffer != nil {
						active = true
					}
					glob.CMSBufferLock.Unlock()

					//If buffer is active, sleep and wait for it to fill up
					if active {
						time.Sleep(constants.CMSRate)

						//Waited for buffer to fill up, grab and clear buffers
						glob.CMSBufferLock.Lock()
						lcopy := glob.CMSBuffer
						glob.CMSBuffer = nil
						glob.CMSBufferLock.Unlock()

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

		//**********************************
		//Pause on connect
		//**********************************
		go func() {
			for glob.ServerRunning {

				time.Sleep(5 * time.Second)
				tn := time.Now()

				if cfg.Local.SlowConnect.SlowConnect {

					glob.ConnectPauseLock.Lock()

					if glob.ConnectPauseTimer > 0 {
						if tn.Unix()-glob.ConnectPauseTimer >= 120 {
							glob.ConnectPauseTimer = 0
							glob.ConnectPauseCount = 0

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

					glob.ConnectPauseLock.Unlock()

				}
			}
		}()

		//**********************************
		//Read and write database regularly
		//**********************************
		go func() {
			return
			for glob.ServerRunning {

				fact.LoadPlayers()
				fact.WritePlayers()
				time.Sleep(time.Minute * 15)

			}
		}()

		//*******************************
		//Save database, if marked dirty
		//*******************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(1 * time.Second)

				glob.PlayerListDirtyLock.Lock()

				if glob.PlayerListDirty {
					glob.PlayerListDirty = false
					//Prevent recursive lock
					go func() {
						botlog.DoLog("Database marked dirty, saving.")
						fact.WritePlayers()
					}()
					//Sleep for a few seconds after writing.
					time.Sleep(10 * time.Second)
				}
				glob.PlayerListDirtyLock.Unlock()
			}
		}()

		//********************************************
		//Save database, if last seen is marked dirty
		//********************************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(5 * time.Minute)
				glob.PlayerListSeenDirtyLock.Lock()

				if glob.PlayerListSeenDirty {
					glob.PlayerListSeenDirty = false

					//Prevent recursive lock
					go func() {
						botlog.DoLog("Database last seen flagged, saving.")
						fact.WritePlayers()
					}()
				}
				glob.PlayerListSeenDirtyLock.Unlock()
			}
		}()

		//***********************************
		//Database file modifcation watching
		//***********************************
		go fact.WatchDatabaseFile()

		//Read database, if the file was modifed
		go func() {
			updated := false

			for glob.ServerRunning {

				time.Sleep(250 * time.Millisecond)

				//Detect update
				glob.PlayerListUpdatedLock.Lock()
				if glob.PlayerListUpdated {
					updated = true
					glob.PlayerListUpdated = false
				}
				glob.PlayerListUpdatedLock.Unlock()

				if updated {
					updated = false

					botlog.DoLog("Database file modified, loading.")
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

				glob.GuildLock.Lock()

				//Get guild id, if we need it

				if glob.Guild == nil && glob.DS != nil {
					// Attempt to get the guild from the state,
					// If there is an error, fall back to the restapi.
					nguild, err := glob.DS.State.Guild(cfg.Global.DiscordData.GuildID)
					if err != nil {
						nguild, err = glob.DS.Guild(cfg.Global.DiscordData.GuildID)
						if err != nil {
							//botlog.DoLog("Failed to get guild data, will retry...")
							return
						}
					}

					if err != nil {
						botlog.DoLog(fmt.Sprintf("Was unable to get guild data from GuildID: %s", err))

						glob.GuildLock.Unlock()
						continue
					}
					if nguild == nil || err != nil {
						glob.Guildname = constants.Unknown
						botlog.DoLog("Guild data came back nil.")
					} else {

						//Guild found, exit loop
						glob.Guild = nguild
						glob.Guildname = nguild.Name
						botlog.DoLog("Guild data linked.")

						glob.GuildLock.Unlock()
						break
					}
				}

				glob.GuildLock.Unlock()
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
						fact.LogCMS(cfg.Local.ChannelData.ChatID, "No players currently online, performing scheduled reboot.")
						fact.QuitFactorio()
						for x := 0; x < 60 && fact.IsFactRunning(); x++ {
							time.Sleep(time.Second)
						}
						break //We don't need to loop anymore
					}
				}
			}
		}()

		//************************************
		//Eventually give up waiting for Factorio to quit
		//************************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(5 * time.Second)

				timer := fact.GetFactQuitTimer()
				if !timer.IsZero() && time.Since(timer) > (60*time.Second) {
					fact.DoExit()
					break
				}
			}
		}()

		//************************************
		//Bug players if there is an pending update
		//************************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(5 * time.Second)

				if cfg.Local.AutoUpdate {
					if fact.IsFactRunning() && glob.NewVersion != constants.Unknown {
						if fact.GetNumPlayers() > 0 {

							numwarn := fact.GetUpdateWarnCounter()

							//Warn users
							if numwarn < glob.UpdateGraceMinutes {
								msg := fmt.Sprintf("(SYSTEM) Factorio update waiting (%v), please log off as soon as there is a good stopping point, players on the upgraded version will be unable to connect (%vm grace remaining)!", glob.NewVersion, glob.UpdateGraceMinutes-numwarn)
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat [color=red]" + msg + "[/color]")
							}
							time.Sleep(1 * time.Minute)

							//Reboot anyway
							if numwarn > glob.UpdateGraceMinutes {
								msg := "(SYSTEM) Rebooting for Factorio update."
								fact.CMS(cfg.Local.ChannelData.ChatID, msg)
								fact.WriteFact("/cchat [color=red]" + msg + "[/color]")
								fact.SetUpdateWarnCounter(0)
								fact.QuitFactorio()
								for x := 0; x < 60 && fact.IsFactRunning(); x++ {
									time.Sleep(time.Second)
								}

								break //Stop looping
							}
							fact.SetUpdateWarnCounter(numwarn + 1)
						} else {
							fact.SetUpdateWarnCounter(0)
							fact.QuitFactorio()
							for x := 0; x < 60 && fact.IsFactRunning(); x++ {
								time.Sleep(time.Second)
							}
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
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Reboot queued!")
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
							for x := 0; x < 60 && fact.IsFactRunning(); x++ {
								time.Sleep(time.Second)
							}
							fact.DoExit()
						} else {
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "ChatWire is halting.")
							fact.DoExit()
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

				glob.UpdateChannelLock.Lock()
				chname := glob.NewChanName
				oldchname := glob.OldChanName
				glob.UpdateChannelLock.Unlock()

				if oldchname != chname {
					fact.DoUpdateChannelName()
					time.Sleep(time.Minute * 5)
				} else {

					time.Sleep(5 * time.Second)
				}
			}
		}()

		//****************************
		// Force refresh channel names
		//****************************
		go func() {

			for glob.ServerRunning {
				time.Sleep(time.Hour)
				fact.UpdateChannelName()
				fact.DoUpdateChannelName()
			}
		}()

		//****************************
		// Capture man-minutes
		//****************************
		go func() {
			for glob.ServerRunning {
				time.Sleep(time.Minute)
				nump := fact.GetNumPlayers()

				glob.ManMinutesLock.Lock()
				if nump > 0 {
					glob.ManMinutes = (glob.ManMinutes + nump)
				}
				glob.ManMinutesLock.Unlock()
			}
		}()
	}()

	//After starting loops, wait here for process signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	fact.SetAutoStart(false)
	fact.SetBotReboot(false)
	fact.SetQueued(false)
	fact.QuitFactorio()
	for x := 0; x < 60 && fact.IsFactRunning(); x++ {
		time.Sleep(time.Second)
	}
	fact.DoExit()
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
