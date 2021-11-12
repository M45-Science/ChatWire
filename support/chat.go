package support

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"../cfg"
	"../constants"
	"../disc"
	"../fact"
	"../glob"
	"../sclean"
	embed "github.com/Clinet/discordgo-embed"
	"github.com/hpcloud/tail"
)

// IsPatreon checks if user has patreon role
func IsPatreon(id string) bool {
	if id == "" || glob.DS == nil {
		return false
	}
	g := glob.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.Patreon {
						return true
					}
				}
			}
		}
	}
	return false
}

// IsNitro checks if user has nitro role
func IsNitro(id string) bool {
	if id == "" || glob.DS == nil {
		return false
	}
	g := glob.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.Nitro {
						return true
					}
				}
			}
		}
	}
	return false
}

//True, error
func StringToBool(txt string) (bool, bool) {
	if strings.ToLower(txt) == "true" ||
		strings.ToLower(txt) == "t" ||
		strings.ToLower(txt) == "yes" ||
		strings.ToLower(txt) == "y" ||
		strings.ToLower(txt) == "on" ||
		strings.ToLower(txt) == "1" {
		return true, false
	} else if strings.ToLower(txt) == "false" ||
		strings.ToLower(txt) == "f" ||
		strings.ToLower(txt) == "no" ||
		strings.ToLower(txt) == "n" ||
		strings.ToLower(txt) == "off" ||
		strings.ToLower(txt) == "0" {
		return false, false
	}

	return false, true
}

func BoolToString(b bool) string {
	if b {
		return "on"
	} else {
		return "off"
	}
}

// Chat pipes in-game chat to Discord, and handles log events
func Chat() {

	go func() {
		for {

			t, err := tail.TailFile(glob.GameLogName, tail.Config{Follow: true})
			if err != nil {
				log.Println(fmt.Sprintf("An error occurred when attempting to tail logfile %s Details: %s", glob.GameLogName, err))
				fact.DoExit()
			}

			//*****************
			//TAIL LOOP
			//*****************
			for line := range t.Lines {

				//Strip stuff we don't want
				lineText := sclean.StripControlAndSubSpecial(line.Text)

				linelen := len(lineText)
				//Ignore blanks
				if lineText == "" || linelen <= 1 {
					continue
				}

				//Server is alive
				fact.SetFactRunning(true, false)

				if linelen > 8192 {
					//Message too long
					log.Println("Line from factorio was too long.")
					continue
				}

				//***************************************
				//Pre-process lines for quick use
				//This could be optimized,
				//but would be at cost of repeated code
				//***************************************

				//Timecode removal
				trimmed := strings.TrimLeft(lineText, " ")
				words := strings.Split(trimmed, " ")
				numwords := len(words)
				NoTC := constants.Unknown
				NoDS := constants.Unknown

				if numwords > 1 {
					NoTC = strings.Join(words[1:], " ")
				}
				if numwords > 2 {
					NoDS = strings.Join(words[2:], " ")
				}

				//Seperate args -- for use with script output
				linelist := strings.Split(lineText, " ")
				linelistlen := len(linelist)

				//Seperate args, notc -- for use with factorio subsystem output
				notclist := strings.Split(NoTC, " ")
				notclistlen := len(notclist)

				//Seperate args, nods -- for use with normal factorio log output
				nodslist := strings.Split(NoDS, " ")
				nodslistlen := len(nodslist)

				//Lowercase converted
				lowerline := strings.ToLower(lineText)
				lowerlist := strings.Split(lowerline, " ")
				lowerlistlen := len(lowerlist)

				//Decrement every time we see activity, if we see time not progressing, add two
				glob.PausedTicksLock.Lock()
				if glob.PausedTicks > 0 {
					glob.PausedTicks--
				}
				glob.PausedTicksLock.Unlock()

				//********************************
				//FILTERED AREA
				//NO CMD, ESCAPED OR CONSOLE CHAT
				//*********************************
				if !strings.HasPrefix(lineText, "[CMD]") && !strings.HasPrefix(lineText, "~") && !strings.HasPrefix(lineText, "<server>") {

					//*****************
					//NO CHAT AREA
					//*****************
					if !strings.HasPrefix(NoDS, "[CHAT]") {

						//*****************
						//GET FACTORIO TIME, replace me
						//*****************
						if strings.Contains(lowerline, " second") || strings.Contains(lowerline, " minute") || strings.Contains(lowerline, " hour") || strings.Contains(lowerline, " day") {

							day := 0
							hour := 0
							minute := 0
							second := 0

							//TODO
							//We should check, that at least one starts on 2nd word
							if lowerlistlen > 1 {

								for x := 0; x < lowerlistlen; x++ {
									if strings.Contains(lowerlist[x], "day") {
										day, _ = strconv.Atoi(lowerlist[x-1])
									} else if strings.Contains(lowerlist[x], "hour") {
										hour, _ = strconv.Atoi(lowerlist[x-1])
									} else if strings.Contains(lowerlist[x], "minute") {
										minute, _ = strconv.Atoi(lowerlist[x-1])
									} else if strings.Contains(lowerlist[x], "second") {
										second, _ = strconv.Atoi(lowerlist[x-1])
									}
								}
								newtime := fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)

								//Pause detection
								glob.GametimeLock.Lock()
								glob.PausedTicksLock.Lock()

								if glob.LastGametime == glob.Gametime {
									if glob.PausedTicks <= constants.PauseThresh {
										glob.PausedTicks = glob.PausedTicks + 2
									}
								} else {
									glob.PausedTicks = 0
								}
								glob.LastGametime = glob.Gametime
								glob.GametimeString = lowerline
								glob.Gametime = newtime

								glob.PausedTicksLock.Unlock()
								glob.GametimeLock.Unlock()
							}
							//This might block stuff by accident, don't do it
							//continue
						}

						//*****************
						//COMMAND REPORTING
						//*****************
						if strings.HasPrefix(lineText, "[CMD]") {
							log.Println(lineText)
							continue
						}

						//*****************
						//USER REPORT
						//*****************
						if strings.HasPrefix(lineText, "[REPORT]") {
							if linelistlen >= 3 {
								buf := fmt.Sprintf("**USER REPORT:**\nServer: %v, User: %v: Report:\n %v",
									cfg.Local.ServerCallsign+"-"+cfg.Local.Name, linelist[1], strings.Join(linelist[2:], " "))
								fact.CMS(cfg.Global.DiscordData.ReportChannelID, buf)
								log.Println(lineText)
							}
							continue
						}

						//*****************
						//ACCESS
						//*****************
						if strings.HasPrefix(lineText, "[ACCESS]") {
							if linelistlen == 4 {
								//Format:
								//print("[ACCESS] " .. ptype .. " " .. player.name .. " " .. param.parameter)

								ptype := linelist[1]
								pname := linelist[2]
								code := linelist[3]

								//Filter just in case, and so accidental spaces won't ruin passcodes
								code = strings.ReplaceAll(code, ":", "")
								code = strings.ReplaceAll(code, ",", "")
								code = strings.ReplaceAll(code, " ", "")
								code = strings.ReplaceAll(code, "\n", "")
								code = strings.ReplaceAll(code, "\r", "")

								pname = strings.ReplaceAll(pname, ":", "")
								pname = strings.ReplaceAll(pname, ",", "")
								pname = strings.ReplaceAll(pname, " ", "")
								pname = strings.ReplaceAll(pname, "\n", "")
								pname = strings.ReplaceAll(pname, "\r", "")

								codegood := true
								codefound := false
								plevel := 0

								glob.PasswordListLock.Lock()
								for i := 0; i <= glob.PasswordMax && i <= constants.MaxPasswords; i++ {
									if glob.PasswordList[i] == code {
										codefound = true
										//Delete password from list
										glob.PasswordList[i] = ""
										pid := glob.PasswordID[i]
										glob.PasswordID[i] = ""
										glob.PasswordTime[i] = 0

										newrole := ""
										if ptype == "trusted" {
											newrole = cfg.Global.RoleData.Member
											plevel = 1
										} else if ptype == "regular" {
											newrole = cfg.Global.RoleData.Regular
											plevel = 2
										} else if ptype == "admin" {
											newrole = cfg.Global.RoleData.Admin
											plevel = 255
										} else {
											newrole = cfg.Global.RoleData.New
											plevel = 0
										}

										discid := disc.GetDiscordIDFromFactorioName(pname)
										factname := disc.GetFactorioNameFromDiscordID(pid)

										if discid == pid && factname == pname {
											fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This Factorio account, and Discord account are already connected! Setting role, if needed.", pname))
											codegood = true
											//Do not break, process
										} else if discid != "" {
											log.Println(fmt.Sprintf("Factorio user '%s' tried to connect a Discord user, that is already connected to a different Factorio user.", pname))
											fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] That discord user is already connected to a different Factorio user.", pname))
											codegood = false
											continue
										} else if factname != "" {
											log.Println(fmt.Sprintf("Factorio user '%s' tried to connect their Factorio user, that is already connected to a different Discord user.", pname))
											fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This Factorio user is already connected to a different discord user.", pname))
											codegood = false
											continue
										}

										if codegood {
											fact.PlayerSetID(pname, pid, plevel)

											guild := fact.GetGuild()
											if guild != nil {
												errrole, regrole := disc.RoleExists(guild, newrole)

												if !errrole {
													fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("Sorry, there is an error. I couldn't find the Discord role '%s'.", newrole))
													fact.WriteFact(fmt.Sprintf("/cwhisper %s  [SYSTEM] Sorry, there was an internal error, I coudn't find the Discord role '%s' Let the moderators know!", newrole, pname))
													continue
												}

												erradd := disc.SmartRoleAdd(cfg.Global.DiscordData.GuildID, pid, regrole.ID)

												if erradd != nil || glob.DS == nil {
													fact.CMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("Sorry, there is an error. I couldn't assign the Discord role '%s'.", newrole))
													fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, there was an error, coundn't assign role '%s' Let the moderators know!", newrole, pname))
													continue
												}
												fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Registration complete!", pname))
												fact.LogCMS(cfg.Local.ChannelData.ChatID, pname+": Registration complete!")
												continue
											} else {
												log.Println("No guild info.")
												fact.CMS(cfg.Local.ChannelData.ChatID, "Sorry, I couldn't find the guild info!")
												continue
											}
										}
										continue
									}
								} //End of loop
								glob.PasswordListLock.Unlock()
								if !codefound {
									log.Println(fmt.Sprintf("Factorio user '%s', tried to use an invalid or expired code.", pname))
									fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, that code is invalid or expired. Make sure you are entering the code on the correct Factorio server!", pname))
									continue
								}
							} else {
								log.Println("Internal error, [ACCESS] had wrong argument count.")
								continue
							}
							continue
						}

						//***********************
						//CAPTURE ONLINE PLAYERS
						//***********************
						if strings.HasPrefix(lineText, "Online players") {

							if linelistlen > 2 {
								poc := strings.Join(linelist[2:], " ")
								poc = strings.ReplaceAll(poc, "(", "")
								poc = strings.ReplaceAll(poc, ")", "")
								poc = strings.ReplaceAll(poc, ":", "")
								poc = strings.ReplaceAll(poc, " ", "")

								nump, _ := strconv.Atoi(poc)
								fact.SetNumPlayers(nump)

								glob.RecordPlayersLock.Lock()
								if nump > glob.RecordPlayers {
									glob.RecordPlayers = nump

									//New thread, avoid deadlock
									go func() {
										fact.WriteRecord()
									}()

									buf := fmt.Sprintf("**New record!** Players online: %v", glob.RecordPlayers)
									fact.CMS(cfg.Local.ChannelData.ChatID, buf)
									//write to factorio as well
									buf = strings.ReplaceAll(buf, "*", "") //Remove bold
									fact.WriteFact("/cchat " + buf)

								}
								glob.RecordPlayersLock.Unlock()

								fact.UpdateChannelName()
							}
							continue
						}
						//*****************
						//JOIN AREA
						//*****************
						if strings.HasPrefix(NoDS, "[JOIN]") {
							fact.WriteFact("/p o c")

							if nodslistlen > 1 {
								pname := sclean.StripControlAndSubSpecial(nodslist[1])
								glob.NumLoginsLock.Lock()
								glob.NumLogins = glob.NumLogins + 1
								glob.NumLoginsLock.Unlock()
								plevelname := fact.AutoPromote(pname)

								pname = sclean.EscapeDiscordMarkdown(pname)

								buf := fmt.Sprintf("`%-11s` **%s joined**%s", fact.GetGameTime(), pname, plevelname)
								fact.CMS(cfg.Local.ChannelData.ChatID, buf)

								//Give people patreon/nitro tags in-game.
								did := disc.GetDiscordIDFromFactorioName(pname)
								if did != "" {
									if IsPatreon(did) {
										fact.WriteFact(fmt.Sprintf("/regular %s", pname))
										fact.WriteFact(fmt.Sprintf("/patreon %s", pname))
									}
									if IsNitro(did) {
										fact.WriteFact(fmt.Sprintf("/regular %s", pname))
										fact.WriteFact(fmt.Sprintf("/nitro %s", pname))
									}
								}
							}
							continue
						}
						//*****************
						//LEAVE
						//*****************
						if strings.HasPrefix(NoDS, "[LEAVE]") {
							fact.WriteFact("/p o c")

							if nodslistlen > 1 {
								pname := nodslist[1]

								go func(factname string) {
									fact.UpdateSeen(factname)
								}(pname)
							}
							newtime := time.Now()
							savetimer := fact.GetSaveTimer()

							//Only save if time has passed
							if newtime.Sub(savetimer) >= 60 {
								fact.SetSaveTimer()
								fact.SaveFactorio()
							}
							continue
						}

						//*****************
						//MSG AREA
						//*****************
						if strings.HasPrefix(lineText, "[MSG]") {

							if linelistlen > 0 {
								ctext := strings.Join(linelist[1:], " ")

								//Clean strings
								cmess := sclean.StripControlAndSubSpecial(ctext)
								cmess = sclean.EscapeDiscordMarkdown(cmess)
								cmess = sclean.RemoveFactorioTags(cmess)

								if len(cmess) > 500 {
									cmess = fmt.Sprintf("%s...(cut, too long!)", sclean.TruncateString(cmess, 500))
								}

								fact.CMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%-11s` **%s**", fact.GetGameTime(), cmess))
							}

							if linelistlen > 1 {
								trustname := linelist[1]

								if trustname != "" {

									if strings.Contains(lineText, " is now a member!") {
										fact.PlayerLevelSet(trustname, 1)
										fact.AutoPromote(trustname)
										continue
									} else if strings.Contains(lineText, " is now a regular!") {
										fact.PlayerLevelSet(trustname, 2)
										fact.AutoPromote(trustname)
										continue
									} else if strings.Contains(lineText, " moved to Admins group.") {
										fact.PlayerLevelSet(trustname, 255)
										fact.AutoPromote(trustname)
										continue
									} else if strings.Contains(lineText, " to the map!") && strings.Contains(lineText, "Welcome ") {
										btrustname := linelist[2]
										fact.AutoPromote(btrustname)
										continue
									} else if strings.Contains(lineText, " has nil permissions.") {
										fact.AutoPromote(trustname)
										continue
									}
								}
							}
							continue
						}
						//*****************
						//BAN
						//*****************
						if strings.HasPrefix(NoDS, "[BAN]") {

							if nodslistlen > 1 {
								trustname := nodslist[1]

								if strings.Contains(NoDS, "was banned by") {
									fact.PlayerLevelSet(trustname, -1)
									go func() {
										fact.WritePlayers()
									}()
								}

								fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%-11s` %s", fact.GetGameTime(), strings.Join(nodslist[1:], " ")))
							}
							continue
						}
						//*****************
						//(ONLINE)
						//*****************
						//if strings.Contains(lineText, "(online)") {

						//Upgrade or replace this...
						//fact.CMS(cfg.Local.ChannelData.ChatID, lineText)
						//continue
						//}

						//*****************
						//Pause on catch-up
						//*****************
						if cfg.Local.SlowConnect.SlowConnect {

							tn := time.Now()

							if strings.HasPrefix(NoTC, "Info ServerMultiplayerManager") {

								if strings.Contains(lineText, "removing peer") {
									fact.WriteFact("/p o c")

									newtime := time.Now()
									savetimer := fact.GetSaveTimer()

									//Only save if time has passed
									if newtime.Sub(savetimer) >= 60 {
										fact.SetSaveTimer()
										fact.SaveFactorio()
									}

									//Fix for players leaving with no leave message
								} else if strings.Contains(lineText, "oldState(ConnectedLoadingMap) newState(TryingToCatchUp)") {
									if cfg.Local.SlowConnect.ConnectSpeed <= 0.0 {
										fact.WriteFact("/gspeed 0.5")
									} else {
										fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.SlowConnect.ConnectSpeed))
									}

									glob.ConnectPauseLock.Lock()
									glob.ConnectPauseTimer = tn.Unix()
									glob.ConnectPauseCount++
									glob.ConnectPauseLock.Unlock()

								} else if strings.Contains(lineText, "oldState(WaitingForCommandToStartSendingTickClosures) newState(InGame)") {

									glob.ConnectPauseLock.Lock()

									glob.ConnectPauseCount--
									if glob.ConnectPauseCount <= 0 {
										glob.ConnectPauseCount = 0
										glob.ConnectPauseTimer = 0

										if cfg.Local.SlowConnect.DefaultSpeed >= 0.0 {
											fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.SlowConnect.DefaultSpeed))
										} else {
											fact.WriteFact("/gspeed 1.0")
										}
									}

									glob.ConnectPauseLock.Unlock()
								}

							}
						}

						//*****************
						//MAP LOAD
						//*****************
						if strings.HasPrefix(NoTC, "Loading map") {

							//Strip file path
							if notclistlen > 3 {
								fullpath := notclist[2]
								size := notclist[3]
								sizei, _ := strconv.Atoi(size)
								fullpath = strings.Replace(fullpath, ":", "", -1)

								regaa := regexp.MustCompile(`\/.*?\/saves\/`)
								filename := regaa.ReplaceAllString(fullpath, "")

								glob.GameMapLock.Lock()
								glob.GameMapName = filename
								glob.GameMapPath = fullpath
								glob.GameMapLock.Unlock()

								fsize := 0.0
								if sizei > 0 {
									fsize = (float64(sizei) / 1024.0 / 1024.0)
								}

								buf := fmt.Sprintf("Loading map %s (%.2fmb)...", filename, fsize)
								fact.LogCMS(cfg.Local.ChannelData.ChatID, buf)
							} else { //Just in case
								log.Println("Loading map...")
							}
							continue
						}
						//******************
						//RESET MOD MESSAGE
						//******************
						if strings.HasPrefix(NoTC, "Loading mod core") {
							glob.ModLoadLock.Lock()
							glob.ModLoadMessage = nil
							glob.ModLoadString = constants.Unknown
							glob.ModLoadLock.Unlock()
							continue
						}
						//*****************
						//LOADING MOD
						//*****************
						if strings.HasPrefix(NoTC, "Loading mod") && strings.Contains(NoTC, "(data.lua)") &&
							!strings.Contains(NoTC, "settings") && !strings.Contains(NoTC, "base") && !strings.Contains(NoTC, "core") {

							if notclistlen > 4 && glob.DS != nil {

								glob.ModLoadLock.Lock()

								//disabled
								if glob.ModLoadMessage == nil {
									modmess, cerr := glob.DS.ChannelMessageSend(cfg.Local.ChannelData.ChatID, "Loading mods...")
									if cerr != nil {
										log.Println(fmt.Sprintf("An error occurred when attempting to send mod load message. Details: %s", cerr))
										glob.ModLoadMessage = nil
										glob.ModLoadString = constants.Unknown

									} else {
										glob.ModLoadMessage = modmess

										if glob.ModLoadString == constants.Unknown {
											glob.ModLoadString = strings.Join(notclist[2:4], "-")
										}
										_, err := glob.DS.ChannelMessageEdit(cfg.Local.ChannelData.ChatID, glob.ModLoadMessage.ID, "Loading mods: "+glob.ModLoadString)

										if err != nil {
											log.Println(fmt.Sprintf("An error occurred when attempting to edit mod load message. Details: %s", err))
										}
									}
								} else {

									glob.ModLoadString = glob.ModLoadString + ", " + strings.Join(notclist[2:4], "-")
									_, err := glob.DS.ChannelMessageEdit(cfg.Local.ChannelData.ChatID, glob.ModLoadMessage.ID, "Loading mods: "+glob.ModLoadString)
									if err != nil {
										log.Println(fmt.Sprintf("An error occurred when attempting to edit mod load message. Details: %s", err))
									}
								}

								glob.ModLoadLock.Unlock()
							}
							continue
						}

						//*****************
						//GOODBYE
						//*****************
						if strings.HasPrefix(NoTC, "Goodbye") {
							//Factorio has completety closed, stop quit timer!
							fact.StopFactQuitTimer()

							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio is now offline.")
							fact.SetFactorioBooted(false)
							fact.SetFactRunning(false, false)
							continue
						}

						//*****************
						//READY MESSAGE
						//*****************
						// 5.164 Info RemoteCommandProcessor.cpp:131: Starting RCON interface at IP ADDR:({0.0.0.0:9100})
						if strings.HasPrefix(NoTC, "Info RemoteCommandProcessor") && strings.Contains(NoTC, "Starting RCON interface") {
							fact.SetFactorioBooted(true)
							fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio "+glob.FactorioVersion+" is now online.")
							fact.WriteFact("/p o c")

							fact.WriteFact("/cname " + strings.ToUpper(cfg.Local.ServerCallsign+"-"+cfg.Local.Name))

							//Config new-users restrictions
							if cfg.Local.SoftModOptions.RestrictMode {
								fact.WriteFact("/restrict on")
							} else {
								fact.WriteFact("/restrict off")
							}

							//Config friendly fire
							if cfg.Local.SoftModOptions.FriendlyFire {
								fact.WriteFact("/friendlyfire on")
							} else {
								fact.WriteFact("/friendlyfire off")
							}

							//Config reset-interval
							if cfg.Local.ResetScheduleText != "" {
								fact.WriteFact("/resetint " + cfg.Local.ResetScheduleText)
							}
							if cfg.Local.SoftModOptions.CleanMapOnBoot {
								fact.LogCMS(cfg.Local.ChannelData.ChatID, "Cleaning map.")
								fact.WriteFact("/cleanmap")
							}
							if cfg.Local.DefaultUPSRate > 0 && cfg.Local.DefaultUPSRate < 1000 {
								fact.WriteFact("/aspeed " + fmt.Sprintf("%d", cfg.Local.DefaultUPSRate))
								fact.LogCMS(cfg.Local.ChannelData.ChatID, "Game UPS set to "+fmt.Sprintf("%d", cfg.Local.DefaultUPSRate)+"hz.")
							}
							if cfg.Local.DisableBlueprints {
								fact.WriteFact("/blueprints off")
								fact.LogCMS(cfg.Local.ChannelData.ChatID, "Blueprints disabled.")
							}
							continue
						}

						//*********************
						//GET FACTORIO VERSION
						//*********************
						if strings.HasPrefix(NoTC, "Loading mod base") {
							if notclistlen > 3 {
								glob.FactorioVersion = notclist[3]
							}
							continue
						}

						//**********************
						//CAPTURE SAVE MESSAGES
						//**********************
						if strings.HasPrefix(NoTC, "Info AppManager") && strings.Contains(NoTC, "Saving to") {
							if !cfg.Local.HideAutosaves {
								savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
								savmatch := savreg.FindStringSubmatch(NoTC)
								if len(savmatch) > 1 {
									fact.LogCMS(cfg.Local.ChannelData.ChatID, "💾 "+savmatch[1])
								}
							}
							fact.SetSaveTimer()
							continue
						}
						//**************************
						//CAPTURE MAP NAME, ON EXIT
						//**************************
						if strings.HasPrefix(NoTC, "Info MainLoop") && strings.Contains(NoTC, "Saving map as") {

							//Strip file path
							if notclistlen > 5 {
								fullpath := notclist[5]
								regaa := regexp.MustCompile(`\/.*?\/saves\/`)
								filename := regaa.ReplaceAllString(fullpath, "")
								filename = strings.Replace(filename, ":", "", -1)

								glob.GameMapLock.Lock()
								glob.GameMapName = filename
								glob.GameMapPath = fullpath
								glob.GameMapLock.Unlock()

								fact.LogCMS(cfg.Local.ChannelData.ChatID, "Map saved as: "+filename)

							}
							continue
						}
						//*****************
						//CAPTURE DESYNC
						//*****************
						if strings.HasPrefix(NoTC, "Info") {
							if strings.Contains(NoTC, "DesyncedWaitingForMap") {
								log.Println("desync: " + NoTC)
								continue
							}
						}
						//*****************
						//CAPTURE CRASHES
						//*****************
						if strings.HasPrefix(NoTC, "Error") {
							fact.CMS(cfg.Local.ChannelData.ChatID, "error: "+NoTC)
							//Lock error
							if strings.Contains(NoTC, "Couldn't acquire exclusive lock") {
								fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio is already running.")
								fact.SetAutoStart(false)
								fact.SetFactorioBooted(false)
								fact.SetFactRunning(false, true)
								continue
							}
							//Mod Errors
							if strings.Contains(NoTC, "caused a non-recoverable error.") {
								fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio crashed.")
								fact.SetFactorioBooted(false)
								fact.SetFactRunning(false, true)
								continue
							}
							//Stack traces
							if strings.Contains(NoTC, "Hosting multiplayer game failed") {
								fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio was unable to launch.")
								fact.SetAutoStart(false)
								fact.SetFactorioBooted(false)
								fact.SetFactRunning(false, true)
								continue
							}
							//level.dat
							if strings.Contains(NoTC, "level.dat not found.") {
								fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
								fact.SetAutoStart(false)
								fact.SetFactorioBooted(false)
								fact.SetFactRunning(false, true)
								continue
							}
							//Stack traces
							if strings.Contains(NoTC, "Unexpected error occurred.") {
								fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio crashed.")
								fact.SetFactorioBooted(false)
								fact.SetFactRunning(false, true)
								continue
							}
							//Multiplayer manger
							if strings.Contains(NoTC, "MultiplayerManager failed:") {
								if strings.Contains(NoTC, "info.json not found") {
									fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
									fact.SetAutoStart(false)
									fact.SetFactorioBooted(false)
									fact.SetFactRunning(false, true)
									continue
								}
								//Corrupt savegame
								if strings.Contains(NoTC, "Closing file") {
									glob.GameMapLock.Lock()
									path := glob.GameMapPath
									glob.GameMapLock.Unlock()

									var tempargs []string
									tempargs = append(tempargs, "-f")
									tempargs = append(tempargs, path)

									out, errs := exec.Command(cfg.Global.PathData.RMPath, tempargs...).Output()

									if errs != nil {
										log.Println(fmt.Sprintf("Unabled to delete corrupt savegame. Details:\nout: %v\nerr: %v", string(out), errs))
										fact.SetAutoStart(false)
										fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
									} else {
										log.Println("Deleted corrupted savegame.")
										fact.CMS(cfg.Local.ChannelData.ChatID, "Save-game corrupted, performing roll-back.")
									}

									fact.SetFactorioBooted(false)
									fact.SetFactRunning(false, true)
									continue
								}
								if strings.Contains(NoTC, "Failed to reach auth server.") {
									fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to connect to auth.factorio.com. Server will not show up in factorio server list, reboot to re-attempt.")
									continue
								}
							}
							continue
						}

					}
					//***********************
					//FACTORIO CHAT MESSAGES
					//***********************
					if strings.HasPrefix(NoDS, "[CHAT]") || strings.HasPrefix(NoDS, "[SHOUT]") {

						if nodslistlen > 1 {
							nodslist[1] = strings.Replace(nodslist[1], ":", "", -1)
							pname := nodslist[1]

							if pname != "<server>" {

								cmess := strings.Join(nodslist[2:], " ")
								cmess = sclean.StripControlAndSubSpecial(cmess)
								cmess = sclean.EscapeDiscordMarkdown(cmess)
								cmess = sclean.RemoveFactorioTags(cmess)

								if len(cmess) > 500 {
									cmess = fmt.Sprintf("%s**(message cut, too long!)**", sclean.TruncateString(cmess, 500))
								}

								if cmess == "" {
									continue
								}

								//Yeah, on different thread please.
								go func(ptemp string) {
									fact.UpdateSeen(ptemp)
								}(pname)

								did := disc.GetDiscordIDFromFactorioName(pname)
								dname := disc.GetNameFromID(did, false)
								avatar := disc.GetDiscordAvatarFromId(did, 64)
								factname := sclean.StripControlAndSubSpecial(pname)
								factname = sclean.TruncateString(factname, 25)

								fbuf := ""
								//Filter Factorio names

								factname = sclean.StripControlAndSubSpecial(factname)
								factname = sclean.EscapeDiscordMarkdown(factname)
								if dname != "" {
									fbuf = fmt.Sprintf("`%-11s` **%s**: %s", fact.GetGameTime(), factname, cmess)
								} else {
									fbuf = fmt.Sprintf("`%-11s` %s: %s", fact.GetGameTime(), factname, cmess)
								}

								//Remove all but letters
								filter, _ := regexp.Compile("[^a-zA-Z]+")

								//Name to lowercase
								dnamelower := strings.ToLower(dname)
								fnamelower := strings.ToLower(pname)

								//Reduce to letters only
								dnamereduced := filter.ReplaceAllString(dnamelower, "")
								fnamereduced := filter.ReplaceAllString(fnamelower, "")

								//If we find discord name, and discord name and factorio name don't contain the same name
								if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {
									//Slap data into embed format.
									myembed := embed.NewEmbed().
										SetAuthor("@"+dname, avatar).
										SetDescription(fbuf).
										MessageEmbed

									//Send it off!
									err := disc.SmartWriteDiscordEmbed(cfg.Local.ChannelData.ChatID, myembed)
									if err != nil {
										//On failure, send normal message
										log.Println("Failed to send chat embed.")
									} else {
										//Stop if succeeds
										continue
									}
								}
								fact.CMS(cfg.Local.ChannelData.ChatID, fbuf)
							}
							continue
						}
						continue
					}
					//*****************
					//END CHAT
					//*****************
				}
				//*****************
				//END FILTERED
				//*****************

				//*****************
				//"/online"
				//*****************
				if strings.HasPrefix(lineText, "~") {
					if strings.Contains(lineText, "Online:") {
						fact.CMS(cfg.Local.ChannelData.ChatID, "`"+lineText+"`")
						continue
					}
				}

			}
			//*****************
			//END TAIL LOOP
			//*****************
		}
	}()
}
