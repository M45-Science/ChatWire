package support

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"

	embed "github.com/Clinet/discordgo-embed"
)

// IsPatreon checks if user has patreon role
func IsPatreon(id string) bool {
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.PatreonRoleID {
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
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.NitroRoleID {
						return true
					}
				}
			}
		}
	}
	return false
}

/* Convert string to bool */
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

/* Bool to sting */
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
		for glob.ServerRunning {
			if fact.GameBuffer != nil {
				reader := bufio.NewScanner(fact.GameBuffer)
				time.Sleep(time.Millisecond * 100)
				for reader.Scan() {
					line := reader.Text()
					line = strings.TrimSuffix(line, "\r")
					line = strings.TrimSuffix(line, "\n")
					ll := len(line)
					if ll <= 1 {
						continue
					}
					//Server is alive
					fact.SetFactRunning(true, false)

					//***************************************
					//Pre-process lines for quick use
					//This could be optimized,
					//but would be at cost of repeated code
					//***************************************

					//Timecode removal
					trimmed := strings.TrimLeft(line, " ")
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

					//Separate args -- for use with script output
					linelist := strings.Split(line, " ")
					linelistlen := len(linelist)

					//Separate args, notc -- for use with factorio subsystem output
					notclist := strings.Split(NoTC, " ")
					notclistlen := len(notclist)

					//Separate args, nods -- for use with normal factorio log output
					nodslist := strings.Split(NoDS, " ")
					nodslistlen := len(nodslist)

					//Lowercase converted
					lowerline := strings.ToLower(line)
					lowerlist := strings.Split(lowerline, " ")
					lowerlistlen := len(lowerlist)

					//Decrement every time we see activity, if we see time not progressing, add two
					fact.PausedTicksLock.Lock()
					if fact.PausedTicks > 0 {
						fact.PausedTicks--
					}
					fact.PausedTicksLock.Unlock()

					//********************************
					//FILTERED AREA
					//NO ESCAPED OR CONSOLE CHAT
					//*********************************
					if !strings.HasPrefix(line, "~") && !strings.HasPrefix(line, "<server>") {

						//*****************
						//NO CHAT AREA
						//*****************
						if !strings.HasPrefix(NoDS, "[CHAT]") && !strings.HasPrefix(NoDS, "[SHOUT]") && !strings.HasPrefix(line, "[CMD]") {

							//*****************
							//GET FACTORIO TIME
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

									newtime := constants.Unknown
									if day > 0 {
										newtime = fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)
									} else if hour > 0 {
										newtime = fmt.Sprintf("%.2d-%.2d-%.2d", day, hour, minute, second)
									} else if minute > 0 {
										newtime = fmt.Sprintf("%.2d-%.2d", day, hour, minute, second)
									} else {
										newtime = fmt.Sprintf("%.2d", day, hour, minute, second)
									}

									//Don't add the time if we are slowed down for players connecting, or paused
									if fact.ConnectPauseTimer == 0 && fact.PausedTicks <= 2 {
										fact.TickHistoryLock.Lock()
										fact.TickHistory = append(fact.TickHistory,
											fact.TickInt{Day: day, Hour: hour, Min: minute, Sec: second})

										//Chop old history
										thl := len(fact.TickHistory) - fact.MaxTickHistory
										if thl > 0 {
											fact.TickHistory = fact.TickHistory[thl:]
										}
										fact.TickHistoryLock.Unlock()
									}

									//Pause detection
									fact.GametimeLock.Lock()
									fact.PausedTicksLock.Lock()

									if fact.LastGametime == fact.Gametime {
										if fact.PausedTicks <= constants.PauseThresh {
											fact.PausedTicks = fact.PausedTicks + 2
										}
									} else {
										fact.PausedTicks = 0
									}
									fact.LastGametime = fact.Gametime
									fact.GametimeString = lowerline
									fact.Gametime = newtime

									fact.PausedTicksLock.Unlock()
									fact.GametimeLock.Unlock()
								}
								//This might block stuff by accident, don't do it
								//continue
							}

							//*****************
							//USER REPORT
							//*****************
							if strings.HasPrefix(line, "[REPORT]") {
								botlog.DoLogGame(line)
								if linelistlen >= 3 {
									buf := fmt.Sprintf("**USER REPORT:**\nServer: %v, User: %v: Report:\n %v",
										cfg.Local.ServerCallsign+"-"+cfg.Local.Name, linelist[1], strings.Join(linelist[2:], " "))
									fact.CMS(cfg.Global.DiscordData.ReportChannelID, buf)
								}
								continue
							}

							//*****************
							//ACCESS
							//*****************
							if strings.HasPrefix(line, "[ACCESS]") {
								if linelistlen == 4 {
									//Format:
									//print("[ACCESS] " .. ptype .. " " .. player.name .. " " .. param.parameter)

									ptype := linelist[1]
									pname := linelist[2]
									code := linelist[3]

									//Filter just in case, and so accidental spaces won't ruin passcodes
									code = strings.ReplaceAll(code, " ", "")
									pname = strings.ReplaceAll(pname, " ", "")

									codegood := true
									codefound := false
									plevel := 0

									glob.PasswordListLock.Lock()
									for i, pass := range glob.PassList {
										if pass.Code == code {
											codefound = true
											//Delete password from list
											pid := pass.DiscID
											delete(glob.PassList, i)

											newrole := ""
											if ptype == "trusted" {
												newrole = cfg.Global.RoleData.MemberRoleName
												plevel = 1
											} else if ptype == "regular" {
												newrole = cfg.Global.RoleData.RegularRoleName
												plevel = 2
											} else if ptype == "admin" {
												newrole = cfg.Global.RoleData.ModeratorRoleName
												plevel = 255
											} else {
												newrole = cfg.Global.RoleData.NewRoleName
												plevel = 0
											}

											discid := disc.GetDiscordIDFromFactorioName(pname)
											factname := disc.GetFactorioNameFromDiscordID(pid)

											if discid == pid && factname == pname {
												fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This Factorio account, and Discord account are already connected! Setting role, if needed.", pname))
												codegood = true
												//Do not break, process
											} else if discid != "" {
												botlog.DoLog(fmt.Sprintf("Factorio user '%s' tried to connect a Discord user, that is already connected to a different Factorio user.", pname))
												fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] That discord user is already connected to a different Factorio user.", pname))
												codegood = false
												continue
											} else if factname != "" {
												botlog.DoLog(fmt.Sprintf("Factorio user '%s' tried to connect their Factorio user, that is already connected to a different Discord user.", pname))
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
														fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("Sorry, there is an error. I could not find the Discord role '%s'.", newrole))
														fact.WriteFact(fmt.Sprintf("/cwhisper %s  [SYSTEM] Sorry, there was an internal error, I could not find the Discord role '%s' Let the moderators know!", newrole, pname))
														continue
													}

													erradd := disc.SmartRoleAdd(cfg.Global.DiscordData.GuildID, pid, regrole.ID)

													if erradd != nil || disc.DS == nil {
														fact.CMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("Sorry, there is an error. I could not assign the Discord role '%s'.", newrole))
														fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, there was an error, could not assign role '%s' Let the moderators know!", newrole, pname))
														continue
													}
													fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Registration complete!", pname))
													fact.LogCMS(cfg.Local.ChannelData.ChatID, pname+": Registration complete!")
													continue
												} else {
													botlog.DoLog("No guild info.")
													fact.CMS(cfg.Local.ChannelData.ChatID, "Sorry, I couldn't find the guild info!")
													continue
												}
											}
											continue
										}
									} //End of loop
									glob.PasswordListLock.Unlock()
									if !codefound {
										botlog.DoLog(fmt.Sprintf("Factorio user '%s', tried to use an invalid or expired code.", pname))
										fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, that code is invalid or expired. Make sure you are entering the code on the correct Factorio server!", pname))
										continue
									}
								} else {
									botlog.DoLog("Internal error, [ACCESS] had wrong argument count.")
									continue
								}
								continue
							}

							//***********************
							//CAPTURE ONLINE PLAYERS
							//***********************
							if strings.HasPrefix(line, "Online players") {

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
								botlog.DoLogGame(line)
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
								botlog.DoLogGame(line)
								fact.WriteFact("/p o c")

								if nodslistlen > 1 {
									pname := nodslist[1]

									go func(factname string) {
										fact.UpdateSeen(factname)
									}(pname)
								}
								continue
							}

							//*****************
							//MSG AREA
							//*****************
							if strings.HasPrefix(line, "[MSG]") {
								botlog.DoLogGame(line)

								if linelistlen > 0 {
									ctext := strings.Join(linelist[1:], " ")

									//Clean strings
									cmess := sclean.StripControlAndSubSpecial(ctext)
									cmess = sclean.EscapeDiscordMarkdown(cmess)
									cmess = sclean.RemoveFactorioTags(cmess)

									if len(cmess) > 500 {
										cmess = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(cmess, 500))
									}

									fact.CMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%-11s` **%s**", fact.GetGameTime(), cmess))
								}

								if linelistlen > 1 {
									trustname := linelist[1]

									if trustname != "" {

										if strings.Contains(line, " is now a member!") {
											fact.PlayerLevelSet(trustname, 1, false)
											fact.AutoPromote(trustname)
											continue
										} else if strings.Contains(line, " is now a regular!") {
											fact.PlayerLevelSet(trustname, 2, false)
											fact.AutoPromote(trustname)
											continue
										} else if strings.Contains(line, " moved to Admins group.") {
											fact.PlayerLevelSet(trustname, 255, false)
											fact.AutoPromote(trustname)
											continue
										} else if strings.Contains(line, " to the map!") && strings.Contains(line, "Welcome ") {
											btrustname := linelist[2]
											fact.AutoPromote(btrustname)
											continue
										} else if strings.Contains(line, " has nil permissions.") {
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
								botlog.DoLogGame(line)

								if nodslistlen > 1 {
									trustname := nodslist[1]

									if strings.Contains(NoDS, "was banned by") {
										fact.PlayerLevelSet(trustname, -1, false)
									}

									fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%-11s` %s", fact.GetGameTime(), strings.Join(nodslist[1:], " ")))
								}
								continue
							}

							//*****************
							//UNBAN
							//*****************
							if strings.HasPrefix(NoDS, "[UNBANNED]") {
								botlog.DoLogGame(line)

								if nodslistlen > 1 {
									trustname := nodslist[1]

									if strings.Contains(NoDS, "was unbanned by") {
										fact.PlayerLevelSet(trustname, 0, false)
									}

									fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%-11s` %s", fact.GetGameTime(), strings.Join(nodslist[1:], " ")))
								}
								continue
							}

							//*****************
							//Pause on catch-up
							//*****************
							if cfg.Local.SlowConnect.SlowConnect {

								tn := time.Now()

								if strings.HasPrefix(NoTC, "Info ServerMultiplayerManager") {

									if strings.Contains(line, "removing peer") {
										fact.WriteFact("/p o c")

										//Fix for players leaving with no leave message
									} else if strings.Contains(line, "oldState(ConnectedLoadingMap) newState(TryingToCatchUp)") {
										if cfg.Local.SlowConnect.ConnectSpeed <= 0.0 {
											fact.WriteFact("/gspeed 0.5")
										} else {
											fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.SlowConnect.ConnectSpeed))
										}

										fact.ConnectPauseLock.Lock()
										fact.ConnectPauseTimer = tn.Unix()
										fact.ConnectPauseCount++
										fact.ConnectPauseLock.Unlock()

									} else if strings.Contains(line, "oldState(WaitingForCommandToStartSendingTickClosures) newState(InGame)") {

										fact.ConnectPauseLock.Lock()

										fact.ConnectPauseCount--
										if fact.ConnectPauseCount <= 0 {
											fact.ConnectPauseCount = 0
											fact.ConnectPauseTimer = 0

											if cfg.Local.SlowConnect.DefaultSpeed >= 0.0 {
												fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.SlowConnect.DefaultSpeed))
											} else {
												fact.WriteFact("/gspeed 1.0")
											}
										}

										fact.ConnectPauseLock.Unlock()
									}

								}
							}

							//*****************
							//MAP LOAD
							//*****************
							if strings.HasPrefix(NoTC, "Loading map") {
								botlog.DoLogGame(line)

								//Strip file path
								if notclistlen > 3 {
									fullpath := notclist[2]
									size := notclist[3]
									sizei, _ := strconv.Atoi(size)
									fullpath = strings.Replace(fullpath, ":", "", -1)

									regaa := regexp.MustCompile(`\/.*?\/saves\/`)
									filename := regaa.ReplaceAllString(fullpath, "")

									fact.GameMapLock.Lock()
									fact.GameMapName = filename
									fact.GameMapPath = fullpath
									fact.LastSaveName = filename
									fact.GameMapLock.Unlock()

									fsize := 0.0
									if sizei > 0 {
										fsize = (float64(sizei) / 1024.0 / 1024.0)
									}

									buf := fmt.Sprintf("Loading map %s (%.2fmb)...", filename, fsize)
									botlog.DoLog(buf)
								} else { //Just in case
									botlog.DoLog("Loading map...")
								}
								continue
							}
							//*****************
							//LOADING MOD
							//*****************
							if strings.HasPrefix(NoTC, "Loading mod") && strings.HasSuffix(NoTC, "(data.lua)") {

								if !strings.Contains(NoTC, "base") && !strings.Contains(NoTC, "core") {
									botlog.DoLogGame(line)

									modName := strings.TrimPrefix(NoTC, "Loading mod ")
									modName = strings.TrimSuffix(modName, " (data.lua)")
									modName = strings.ReplaceAll(modName, " ", "-")
									fact.AddModLoadString(modName)
								}
								continue
							}

							//*****************
							//GOODBYE
							//*****************
							if strings.HasPrefix(NoTC, "Goodbye") {
								botlog.DoLogGame(line)

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
								fact.SetFactRunning(true, false)
								fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio "+fact.FactorioVersion+" is now online.")
								time.Sleep(time.Second)
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
									//fact.LogCMS(cfg.Local.ChannelData.ChatID, "Cleaning map.")
									fact.WriteFact("/cleanmap")
								}
								if cfg.Local.SoftModOptions.DefaultUPSRate > 0 && cfg.Local.SoftModOptions.DefaultUPSRate < 1000 {
									fact.WriteFact("/aspeed " + fmt.Sprintf("%d", cfg.Local.SoftModOptions.DefaultUPSRate))
									//fact.LogCMS(cfg.Local.ChannelData.ChatID, "Game UPS set to "+fmt.Sprintf("%d", cfg.Local.DefaultUPSRate)+"hz.")
								}
								if cfg.Local.SoftModOptions.DisableBlueprints {
									fact.WriteFact("/blueprints off")
									//fact.LogCMS(cfg.Local.ChannelData.ChatID, "Blueprints disabled.")
								}
								if cfg.Local.SoftModOptions.EnableCheats {
									fact.WriteFact("/enablecheats on")
									//fact.LogCMS(cfg.Local.ChannelData.ChatID, "Cheats enabled.")
								}

								//Patreon list
								disc.RoleListLock.Lock()
								if len(disc.RoleList.Patreons) > 0 {
									fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ","))
								}
								if len(disc.RoleList.NitroBooster) > 0 {
									fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ","))
								}
								disc.RoleListLock.Unlock()
								continue
							}

							//*********************
							//FIX LOCKERS
							//*********************
							if strings.Contains(NoTC, "ServerMultiplayerManager") {
								if strings.HasSuffix(NoTC, "changing state from(InGameSavingMap) to(InGame)") {

									fact.LockerLock.Lock()
									fact.LockerDetectStart = time.Now()
									fact.LockerStart = true
									fact.LockerLock.Unlock()

								} else if strings.HasSuffix(NoTC, "oldState(ConnectedWaitingForMap) newState(ConnectedDownloadingMap)") ||
									strings.Contains(NoTC, "Disconnect notification for peer") {

									fact.LockerLock.Lock()
									fact.LockerDetectStart = time.Now()
									fact.LockerStart = false
									fact.LockerLock.Unlock()

								}
							}

							//*********************
							//Announce incoming connections
							//*********************
							if strings.Contains(NoTC, "Queuing ban recommendation check for user ") {
								if numwords > 1 {
									fact.LockerLock.Lock()
									pName := words[numwords-1]
									fact.LastLockerName = pName
									fact.LockerLock.Unlock()
									dmsg := fmt.Sprintf("`%-11s` %v is connecting.", fact.GetGameTime(), pName)
									fmsg := fmt.Sprintf("%v is connecting.", pName)
									fact.WriteFact("/cchat " + fmsg)
									fact.CMS(cfg.Local.ChannelData.ChatID, dmsg)
								}
							}

							//*********************
							//GET FACTORIO VERSION
							//*********************
							if strings.HasPrefix(NoTC, "Loading mod base") {
								botlog.DoLogGame(line)
								if notclistlen > 3 {
									fact.FactorioVersion = notclist[3]
								}
							}

							//**********************
							//CAPTURE SAVE MESSAGES
							//**********************
							if strings.HasPrefix(NoTC, "Info AppManager") && strings.Contains(NoTC, "Saving to") {
								if !cfg.Local.HideAutosaves {
									savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
									savmatch := savreg.FindStringSubmatch(NoTC)
									if len(savmatch) > 1 {
										if !cfg.Local.HideAutosaves {
											buf := fmt.Sprintf("`%-11s` 💾 %s", fact.GetGameTime(), savmatch[1])
											fact.LogCMS(cfg.Local.ChannelData.ChatID, buf)
										}
										fact.LastSaveName = savmatch[1]
									}
								}
								continue
							}
							//**************************
							//CAPTURE MAP NAME, ON EXIT
							//**************************
							if strings.HasPrefix(NoTC, "Info MainLoop") && strings.Contains(NoTC, "Saving map as") {
								botlog.DoLogGame(line)

								//Strip file path
								if notclistlen > 5 {
									fullpath := notclist[5]
									regaa := regexp.MustCompile(`\/.*?\/saves\/`)
									filename := regaa.ReplaceAllString(fullpath, "")
									filename = strings.Replace(filename, ":", "", -1)

									fact.GameMapLock.Lock()
									fact.GameMapName = filename
									fact.GameMapPath = fullpath
									fact.GameMapLock.Unlock()

									botlog.DoLog(fmt.Sprintf("Map saved as: " + filename))
									fact.LastSaveName = filename

								}
								continue
							}
							//*****************
							//CAPTURE DESYNC
							//*****************
							if strings.HasPrefix(NoTC, "Info") {

								if strings.Contains(NoTC, "DesyncedWaitingForMap") {
									botlog.DoLogGame(line)
									botlog.DoLog("desync: " + NoTC)
									continue
								}
							}
							//*****************
							//CAPTURE CRASHES
							//*****************
							if strings.HasPrefix(NoTC, "Error") {
								botlog.DoLogGame(line)

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
										fact.GameMapLock.Lock()
										path := fact.GameMapPath
										fact.GameMapLock.Unlock()

										var tempargs []string
										tempargs = append(tempargs, "-f")
										tempargs = append(tempargs, path)

										out, errs := exec.Command(cfg.Global.PathData.RMPath, tempargs...).Output()

										if errs != nil {
											botlog.DoLog(fmt.Sprintf("Unabled to delete corrupt savegame. Details:\nout: %v\nerr: %v", string(out), errs))
											fact.SetAutoStart(false)
											fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
										} else {
											botlog.DoLog("Deleted corrupted savegame.")
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
							botlog.DoLogGame(line)

							if nodslistlen > 1 {
								nodslist[1] = strings.Replace(nodslist[1], ":", "", -1)
								pname := nodslist[1]

								if pname != "<server>" {

									cmess := strings.Join(nodslist[2:], " ")
									cmess = sclean.StripControlAndSubSpecial(cmess)
									cmess = sclean.EscapeDiscordMarkdown(cmess)
									cmess = sclean.RemoveFactorioTags(cmess)

									if len(cmess) > 500 {
										cmess = fmt.Sprintf("%s**(message cut, too long!)**", sclean.TruncateStringEllipsis(cmess, 500))
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
											botlog.DoLog("Failed to send chat embed.")
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
						//COMMAND REPORTING
						//*****************
						if strings.HasPrefix(line, "[CMD]") {
							botlog.DoLogGame(line)
							continue
						}
					}
					//*****************
					//END FILTERED
					//*****************

					//*****************
					//"/online"
					//*****************
					if strings.HasPrefix(line, "~") {
						botlog.DoLogGame(line)
						if strings.Contains(line, "Online:") {
							fact.CMS(cfg.Local.ChannelData.ChatID, "`"+line+"`")
							continue
						}
					}

				}
			}
		}
	}()
}
