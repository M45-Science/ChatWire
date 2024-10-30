package support

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	embed "github.com/Clinet/discordgo-embed"
	"github.com/dustin/go-humanize"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

/* Protect players from dumb mistakes with registration codes */
func handleIdiots(input *handleData) bool {
	if ProtectIdiots(input.line) {
		buf := "You did not enter that as a command!\nYou have posted your registration code publicly.\nTo protect you, the code has been invalidated.\nPlease try again, and read the directions more carefully."
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, buf)
		return true
	}

	return false
}

func handleDisconnect(input *handleData) bool {

	if strings.HasPrefix(input.noTimecode, "Info ServerMultiplayerManager") {

		if glob.SoftModVersion == constants.Unknown {
			if strings.Contains(input.line, "removing peer") {
				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "A player has disconnected.")
				fact.WriteFact(glob.OnlineCommand)
			}
		}
	}

	return false
}

func handleGameTime(input *handleData) bool {
	/******************************************************
	 * GET FACTORIO TIME
	 * While this is needed for games without our softmod,
	 * we should be using tick count instead.
	 ******************************************************/
	if strings.Contains(input.lowerLine, " second") || strings.Contains(input.lowerLine, " minute") || strings.Contains(input.lowerLine, " hour") || strings.Contains(input.lowerLine, " day") {

		day := 0
		hour := 0
		minute := 0
		second := 0

		if input.lowerListLen > 1 {

			for x := 0; x < input.lowerListLen; x++ {
				if strings.Contains(input.lowerWordList[x], "day") {
					day, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "hour") {
					hour, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "minute") {
					minute, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "second") {
					second, _ = strconv.Atoi(input.lowerWordList[x-1])
				}
			}

			var newtime string
			if day > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)
			} else if hour > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d-%.2d", hour, minute, second)
			} else if minute > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d", minute, second)
			} else {
				newtime = fmt.Sprintf("%.2d", second)
			}

			/* Don't add the time if we are slowed down for players connecting, or paused */
			if fact.SlowConnectTimer == 0 && fact.PausedTicks <= 2 {
				fact.TickHistoryLock.Lock()
				fact.TickHistory = append(fact.TickHistory,
					fact.TickInt{Day: day, Hour: hour, Min: minute, Sec: second})

				/* Chop old tick history */
				thl := len(fact.TickHistory) - fact.MaxTickHistory
				if thl > 0 {
					fact.TickHistory = fact.TickHistory[thl:]
				}
				fact.TickHistoryLock.Unlock()
			}

			if fact.LastGametime == fact.Gametime {
				if fact.PausedTicks <= constants.PauseThresh {
					fact.PausedTicks = fact.PausedTicks + 2
				}
			} else {
				fact.PausedTicks = 0
			}
			fact.LastGametime = fact.Gametime
			fact.GametimeString = input.lowerLine
			fact.Gametime = newtime
		}
	}
	/* This might block input by accident, don't do it */
	return false
}

func handlePlayerReport(input *handleData) bool {
	/******************
	 * Player REPORT
	 ******************/
	if strings.HasPrefix(input.line, "[REPORT]") {
		cwlog.DoLogGame(input.line)
		if input.wordListLen >= 3 {
			pingStr := ""
			if cfg.Global.Discord.SusPingRole != "" {
				pingStr = fmt.Sprintf("<@&%v>", cfg.Global.Discord.SusPingRole)
			}
			buf := ""
			if cfg.GetGameLogURL() == "" {
				buf = fmt.Sprintf("Server: %v, Reporter: %v: Report:\n %v\n%v",
					cfg.Local.Callsign+"-"+cfg.Local.Name, input.wordList[1], strings.Join(input.wordList[2:], " "), pingStr)
			} else {
				buf = fmt.Sprintf("Server: %v, Reporter: %v: Report:\n %v\nLog: %v\n%v",
					cfg.Local.Callsign+"-"+cfg.Local.Name, input.wordList[1], strings.Join(input.wordList[2:], " "), cfg.GetGameLogURL(), pingStr)
			}
			fact.LogGameCMS(true, cfg.Global.Discord.ReportChannel, buf)
		}
		return true
	}

	return false
}

func handlePlayerRegister(input *handleData) bool {
	/******************
	 * ACCESS
	 ******************/
	if strings.HasPrefix(input.line, "[ACCESS]") {
		if input.wordListLen >= 4 {
			/* Format:
			 * print("[ACCESS] " .. ptype .. " " .. player.name .. " " .. param.parameter) */

			ptype := input.wordList[1]
			pname := input.wordList[2]
			code := strings.Join(input.wordList[3:input.wordListLen], "")

			/* Filter non-letters */
			inputCode := sclean.AlphaOnly(code)

			codegood := true
			codefound := false
			plevel := 0

			glob.PasswordListLock.Lock()
			for i, pass := range glob.PassList {

				/* Case insensitive match */
				chkCode := sclean.AlphaOnly(pass.Code)
				if strings.EqualFold(chkCode, inputCode) {

					codefound = true
					/* Delete password from list */
					pid := pass.DiscID
					delete(glob.PassList, i)

					newrole := ""
					if strings.EqualFold(ptype, "member") {
						newrole = cfg.Global.Discord.Roles.Member
						plevel = 1
					} else if strings.EqualFold(ptype, "regular") {
						newrole = cfg.Global.Discord.Roles.Regular
						plevel = 2
					} else if strings.EqualFold(ptype, "veteran") {
						newrole = cfg.Global.Discord.Roles.Veteran
						plevel = 3
					} else if strings.EqualFold(ptype, "moderator") {
						newrole = cfg.Global.Discord.Roles.Moderator
						plevel = 255
					} else {
						newrole = cfg.Global.Discord.Roles.New
						plevel = 0
					}

					discid := disc.GetDiscordIDFromFactorioName(pname)
					factname := disc.GetFactorioNameFromDiscordID(pid)

					if !strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) {
						/* Some people just can't be bothered to read two short lines of text. */
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', tried to register... but can't read the directions.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This is not the correct server for entering registration codes! You need to connect to %v-%v to use that command. Please read the directions more carefully...",
							pname, cfg.Global.GroupName, cfg.Global.PrimaryServer))
						return true
					}

					if strings.EqualFold(discid, pid) && strings.EqualFold(factname, pname) {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', wants to register a few times... just to be sure.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This factorio user, and discord user are already connected! You do not need to re-register...", pname))
						codegood = true
						/* Do not break, process */
					} else if discid != "" && discid != "0" {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', tried to register a discord user that is already registered to different factorio player.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] That discord user is already connected to a different factorio user... Unable to complete registration.", pname))
						codegood = false
						continue
					} else if factname != "" {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', tried to register a factorio user that is already registered to a different discord user.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This factorio user is already connected to a different discord user... Unable to complete registration.", pname))
						codegood = false
						continue
					}

					if codegood {
						fact.PlayerSetID(pname, pid, plevel)

						guild := disc.Guild
						if guild != nil && disc.DS != nil {
							errrole, regrole := disc.RoleExists(guild, newrole)

							if !errrole {
								fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Register: Can not find role '%v'. Requested for user '%v'", newrole, pname))
								fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, there was an internal error. I could not find the discord role '%s'. The moderatators will be informed of the issue.", pname, newrole))
								continue
							}

							erradd := disc.SmartRoleAdd(cfg.Global.Discord.Guild, pid, regrole.ID)

							if erradd != nil || disc.DS == nil {
								fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Register: Could not assign role '%v'. Requested for user '%v'.", newrole, pname))
								fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, there was an internal error. I could not assign discord role '%s'. The moderatators will be informed of the issue.", pname, newrole))
								continue
							}
							fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Registration complete!", pname))
							fact.LogGameCMS(true, cfg.Global.Discord.ReportChannel, fmt.Sprintf("Registered player: %v", pname))
							continue
						} else {
							fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, I couldn't find the discord guild info! The moderators will be informed of the issue.", pname))
							fact.LogCMS(cfg.Global.Discord.ReportChannel, "Register: Unable to get discord guild info!")
							continue
						}
					}
					continue
				}
			} /* End of loop */
			glob.PasswordListLock.Unlock()
			if !codefound {
				cwlog.DoLogCW("Register: factorio player '%s' tried to use an invalid or expired code.", pname)
				fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, that code is invalid or expired.", pname))
				return true
			}
		} else {
			cwlog.DoLogCW("Internal error, [ACCESS] had wrong argument count.")
			return true
		}
		return true
	}
	return false
}

func handleOnlinePlayers(input *handleData) bool {
	/* ********************************************************
	 * CAPTURE ONLINE PLAYERS
	 * Only used for servers that are not using our soft mod
	 **********************************************************/
	if strings.HasPrefix(input.line, "Online players") {

		if input.wordListLen > 2 {
			poc := strings.Join(input.wordList[2:], " ")
			poc = strings.ReplaceAll(poc, "(", "")
			poc = strings.ReplaceAll(poc, ")", "")
			poc = strings.ReplaceAll(poc, ":", "")
			poc = strings.ReplaceAll(poc, " ", "")

			nump, _ := strconv.Atoi(poc)
			fact.NumPlayers = (nump)

			fact.UpdateChannelName()
		}
		return true
	}
	return false
}

func handlePlayerJoin(input *handleData) bool {
	/******************
	 * JOIN AREA
	 *****************/
	if strings.HasPrefix(input.noDatestamp, "[JOIN]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			pname := sclean.UnicodeCleanup(input.noDatestampList[1])
			banlist.CheckBanList(pname)
			plevelname := fact.AutoPromote(pname, false, true)

			pname = sclean.EscapeDiscordMarkdown(pname)

			buf := fmt.Sprintf("`%v` **%s joined**%s", fact.Gametime, pname, plevelname)

			/* If softmod is active, handle pause on connect */
			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted {

				glob.PausedLock.Lock()
				if glob.PausedForConnect {
					if strings.EqualFold(glob.PausedFor, pname) {
						glob.PausedForConnect = false
						glob.PausedFor = ""
						glob.PausedConnectAttempt = false
						fact.WriteFact(
							fmt.Sprintf("/gspeed %0.2f", cfg.Local.Options.Speed))
						buf = buf + " (Unpausing game)"
					}
				}
				glob.PausedLock.Unlock()
			}

			fact.CMS(cfg.Local.Channel.ChatChannel, buf)

			/* Update softmod map schedule */
			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted {

				fact.UpdateScheduleDesc()
				if fact.TillReset != "" && cfg.Local.Options.Schedule != "" {
					fact.WriteFact("/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")")
				} else {
					fact.WriteFact("/resetdur")
				}

				/* Give people patreon/nitro tags in-game. */
				did := disc.GetDiscordIDFromFactorioName(pname)
				if did != "" {
					if IsPatreon(did) {
						fact.WriteFact(fmt.Sprintf("/patreon %s", pname))
					}
					if IsNitro(did) {
						fact.WriteFact(fmt.Sprintf("/nitro %s", pname))
					}
				}
			}

			fact.WriteFact(glob.OnlineCommand)
		}
		return true
	}
	return false
}

func handlePlayerLeave(input *handleData) bool {
	/******************
	 * LEAVE
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[LEAVE]") &&
		/* Suppress quit messages from map load */
		fact.FactorioBooted && fact.FactIsRunning {

		cwlog.DoLogGame(input.noDatestamp)

		/* Mark as seen, async */
		if input.noDatestampListLen > 1 {
			pname := input.noDatestampList[1]

			/* Show quit if there is no soft-mod */
			if glob.SoftModVersion == constants.Unknown {
				buf := fmt.Sprintf("%v left.", pname)
				fact.WriteFact(glob.OnlineCommand)
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)

			}

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(pname)
		}
		return true
	}
	return false
}

func handleActMsg(input *handleData) bool {
	/******************
	 * ACT AREA
	 * Used for logs, and to attempt to warn of potential griefing
	 ******************/

	if strings.HasPrefix(input.line, "[ACT]") || strings.HasPrefix(input.line, "[TODO]") || strings.HasPrefix(input.line, "[ERROR]") {

		cwlog.DoLogGame(input.line)
		if input.wordListLen > 2 {

			pname := input.wordList[1]
			action := input.wordList[2]

			words := input.wordList[3:]
			numWords := len(words) - 1

			if pname == "" {
				return true
			}

			/* Mark as seen, async */
			go fact.UpdateSeen(pname)
			if pname != "" {

				p := disc.GetPlayerDataFromName(pname)
				if p != nil && p.Name != "" {
					glob.PlayerListLock.Lock() //lock db
					defer glob.PlayerListLock.Unlock()

					if p.Level < 2 {

						if strings.Contains(action, "placed-ghost") {
							p.SusScore -= 2
						} else if strings.Contains(action, "mined-ghost") {
							p.SusScore -= 1
						} else if strings.Contains(action, "placed") {
							p.SusScore--
						} else if strings.Contains(action, "mined") {
							p.SusScore++
						} else if strings.Contains(action, "deconstructing") {
							if numWords > 3 {
								areaString := words[4]
								areaClean := strings.ReplaceAll(areaString, "sq", "")
								area, _ := strconv.Atoi(areaClean)
								p.SusScore += int64(area / 200)
							}
						}

						thresh := int64(constants.SusWarningThresh)
						if p.Level > 0 {
							thresh += 1000
						}
						if p.SusScore > thresh {

							if time.Since(glob.LastSusWarning) > time.Minute*constants.SusWarningInterval {
								glob.LastSusWarning = time.Now()

								if !cfg.Global.Options.ShutupSusWarn {
									pingStr := ""
									if cfg.Global.Discord.SusPingRole != "" {
										pingStr = fmt.Sprintf("<@&%v>", cfg.Global.Discord.SusPingRole)
									}
									sbuf := fmt.Sprintf("*WARNING*: Player: '%v': Possible suspicious activity!)\n%v", pname, cfg.GetGameLogURL())

									fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, sbuf)

									sbuf = cfg.Global.GroupName + "-" + cfg.Local.Callsign + ": " + cfg.Local.Name + ": " + sbuf

									if pingStr != "" {
										reportMsg := fmt.Sprintf("%v\n%v", sbuf, pingStr)
										fact.CMS(cfg.Global.Discord.ReportChannel, reportMsg)
									} else {
										fact.CMS(cfg.Global.Discord.ReportChannel, sbuf)
									}
								}

								p.SusScore = 0
							}
						}
					} else {
						p.SusScore = 0
					}
				}
			}
		}
		return true
	}

	return false
}

func handleSoftModMsg(input *handleData) bool {
	/******************
	 * MSG AREA
	 ******************/
	if strings.HasPrefix(input.line, "[MSG]") {
		cwlog.DoLogCW(input.line)

		if input.wordListLen > 0 {
			ctext := strings.Join(input.wordList[1:], " ")

			/* Clean strings */
			cmess := sclean.UnicodeCleanup(ctext)
			cmess = sclean.EscapeDiscordMarkdown(cmess)
			cmess = sclean.RemoveFactorioTags(cmess)

			if len(cmess) > 500 {
				cmess = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(cmess, 500))
			}

			if strings.HasPrefix(cmess, "Research") {
				if cfg.Local.Options.HideResearch {
					return true
				}
			}

			fact.CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` **%s**", fact.Gametime, cmess))
		}

		if input.wordListLen > 1 {
			trustname := input.wordList[1]

			if trustname != "" {

				if strings.Contains(input.line, " is now a member!") {
					fact.PlayerLevelSet(trustname, 1, false)
					fact.AutoPromote(trustname, false, false)
					return true
				} else if strings.Contains(input.line, " is now a regular!") {
					fact.PlayerLevelSet(trustname, 2, false)
					fact.AutoPromote(trustname, false, false)
					return true
				} else if strings.Contains(input.line, " is now reset!") {
					fact.PlayerLevelSet(trustname, 0, false)
					fact.AutoPromote(trustname, false, false)
					return true
				} else if strings.Contains(input.line, " moved to moderators group") {
					fact.PlayerLevelSet(trustname, 255, false)
					fact.AutoPromote(trustname, false, false)
					return true
				} else if strings.Contains(input.line, " has nil permissions.") {
					fact.AutoPromote(trustname, false, false)
					return true
				}
			}
		}
		return true
	}
	return false

}

func handleMapLoad(input *handleData) bool {
	/******************
	 * MAP LOAD
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Loading map") {
		cwlog.DoLogCW(input.noTimecode)

		/* Strip file path */
		if input.noTimecodeListLen > 3 {
			fullpath := input.noTimecodeList[2]
			size := input.noTimecodeList[3]
			sizei, _ := strconv.Atoi(size)
			fullpath = strings.Replace(fullpath, ":", "", -1)

			regaa := regexp.MustCompile(`\/.*?\/saves\/`)
			filename := regaa.ReplaceAllString(fullpath, "")

			fact.GameMapName = filename
			fact.GameMapPath = fullpath
			fact.LastSaveName = filename

			cwlog.DoLogCW("Loading map %s (%v)...", filename, humanize.Bytes(uint64(sizei)))
		} else { /* Just in case */
			cwlog.DoLogCW("Loading map...")
		}
		return true
	}
	return false
}

func handleBan(input *handleData) bool {
	/******************
	 * BAN
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[BAN]") {

		glob.PlayerListWriteLock.Lock()
		defer glob.PlayerListWriteLock.Unlock()

		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			trustname := input.noDatestampList[1]

			if strings.Contains(input.noDatestamp, "was banned by") {

				if strings.Contains(input.noDatestamp, "Reason") {

					reasonList := strings.Split(input.noDatestamp, "Reason: ")
					fact.PlayerSetBanReason(trustname, reasonList[1], false)
				} else {
					fact.PlayerLevelSet(trustname, -1, false)
				}
			}

			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.Gametime, strings.Join(input.noDatestampList[1:], " ")))
			fact.WriteFact(glob.OnlineCommand)
		}
		return true
	}
	return false
}

func handleSVersion(input *handleData) bool {
	/******************
	 * SVERSION
	 ******************/
	if strings.HasPrefix(input.line, "[SVERSION]") {
		cwlog.DoLogCW(input.line)

		if input.wordListLen > 0 {
			glob.SoftModVersion = input.wordList[1]
			glob.OnlineCommand = constants.SoftModOnlineCMD
			cwlog.DoLogCW("Softmod detected: " + glob.SoftModVersion)
			ConfigSoftMod()
		}
		return true
	}
	return false
}

func handleUnBan(input *handleData) bool {
	/******************
	 * UNBAN
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[UNBANNED]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			trustname := input.noDatestampList[1]

			if strings.Contains(input.noDatestamp, "was unbanned by") {
				if fact.PlayerLevelGet(trustname, true) < 0 {
					fact.PlayerLevelSet(trustname, 0, false)
				}
			}

			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.Gametime, strings.Join(input.noDatestampList[1:], " ")))
		}
		return true
	}
	return false
}

func handleFactGoodbye(input *handleData) bool {
	/******************
	 * GOODBYE
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Goodbye") {
		cwlog.DoLogCW("Factorio has closed.")
		fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "Factorio is now offline.")
		fact.FactorioBooted = false
		fact.FactorioBootedAt = time.Time{}
		fact.SetFactRunning(false)

		return true
	}
	return false
}

func handleFactReady(input *handleData) bool {
	/*****************
	 * READY MESSAGE
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Info RemoteCommandProcessor") && strings.Contains(input.noTimecode, "Starting RCON interface") {
		fact.WriteAdminlist()
		fact.FactorioBooted = true
		fact.FactorioBootedAt = time.Now()
		fact.SetFactRunning(true)
		fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "Factorio "+fact.FactorioVersion+" is now online.")
		fact.WriteFact("/sversion")
		fact.WriteFact(glob.OnlineCommand)
	}
	return false
}

func handleIncomingAnnounce(input *handleData) bool {
	/********************************
	 * Announce incoming connections
	 ********************************/
	if strings.Contains(input.noTimecode, "Queuing ban recommendation check for user ") {
		if input.trimmedWordsLen > 1 {
			pName := input.trimmedWords[input.trimmedWordsLen-1]

			dmsg := fmt.Sprintf("`%v` %v is connecting.", fact.Gametime, pName)
			fmsg := fmt.Sprintf("%v is connecting.", pName)
			fact.FactChat(fmsg)
			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, dmsg)

			glob.PausedLock.Lock()
			if glob.PausedForConnect {
				if strings.EqualFold(glob.PausedFor, pName) {
					glob.PausedConnectAttempt = true
					fact.WriteFact("/gspeed 0.1")
					msg := "Pausing game, requested by " + pName
					fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, msg)
				}
			}
			glob.PausedLock.Unlock()
			return true
		}
	}
	return false
}

func handleFactVersion(input *handleData) bool {
	/* **********************
	 * GET FACTORIO VERSION
	 ***********************/
	if strings.HasPrefix(input.noTimecode, "Loading mod base") {
		cwlog.DoLogCW(input.noTimecode)
		if input.noTimecodeListLen > 3 {
			fact.FactorioVersion = input.noTimecodeList[3]
		}
	}
	return false
}

func handleSaveMsg(input *handleData) bool {
	/*************************
	 * CAPTURE SAVE MESSAGES
	 *************************/
	if strings.HasPrefix(input.noTimecode, "Info AppManager") && strings.Contains(input.noTimecode, "Saving to") {
		if !cfg.Local.Options.HideAutosaves {
			savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
			savmatch := savreg.FindStringSubmatch(input.noTimecode)
			if len(savmatch) > 1 {
				if !cfg.Local.Options.HideAutosaves {
					buf := fmt.Sprintf("`%v` 💾 %s", fact.Gametime, savmatch[1])
					fact.CMS(cfg.Local.Channel.ChatChannel, buf)
					cwlog.DoLogGame(savmatch[1])
				}
				fact.LastSaveName = savmatch[1]
			}
		}
		return true
	}
	return false
}

func handleExitSave(input *handleData) bool {
	/*****************************
	 * CAPTURE MAP NAME, ON EXIT
	 *****************************/
	if strings.HasPrefix(input.noTimecode, "Info MainLoop") && strings.Contains(input.noTimecode, "Saving map as") {
		cwlog.DoLogCW(input.noTimecode)

		/* Strip file path */
		if input.noTimecodeListLen > 5 {
			//Fix odd filenames with spaces???
			fullpath := strings.Join(input.noTimecodeList[5:], " ")
			regaa := regexp.MustCompile(`\/.*?\/saves\/`)
			filename := regaa.ReplaceAllString(fullpath, "")
			filename = strings.Replace(filename, ":", "", -1)

			/* Increment backup number */
			if cfg.Local.LastSaveBackup < constants.MaxSaveBackups {
				cfg.Local.LastSaveBackup++
			} else {
				cfg.Local.LastSaveBackup = 1
			}

			/* Path for backup save */
			newPath := cfg.Global.Paths.Folders.ServersRoot +
				cfg.Global.Paths.ChatWirePrefix +
				cfg.Local.Callsign + "/" +
				cfg.Global.Paths.Folders.FactorioDir + "/" +
				cfg.Global.Paths.Folders.Saves + "/"

			/* Name for backup save */
			newName := fmt.Sprintf("bak-%v.zip", cfg.Local.LastSaveBackup)

			/* Document save name for archive command */
			fact.GameMapName = filename
			fact.GameMapPath = fullpath

			/* Log actions */
			cwlog.DoLogCW("Map saved as: %v, backup: %v", filename, newName)
			fact.LastSaveName = filename

			/* Open the quit-save */
			from, erra := os.Open(fullpath)
			if erra != nil {

				buf := fmt.Sprintf("An error occurred when attempting to read the save to backup: %s", erra)
				cwlog.DoLogCW(buf)
				//fact.CMS(cfg.Local.Channel.ChatChannel, buf)
				return true
			}
			defer from.Close()

			/* Create the backup file */
			to, errb := os.OpenFile(newPath+newName, os.O_RDWR|os.O_CREATE, 0666)
			if errb != nil {
				buf := fmt.Sprintf("An error occurred when attempting to create the backup save: %s", errb)
				cwlog.DoLogCW(buf)
				return true
			}
			defer to.Close()

			/* Copy data */
			_, errc := io.Copy(to, from)
			if errc != nil {
				cwlog.DoLogCW("An error occurred when attempting to write the backup save: %s", errc)
				return true
			}

			/* Touch old save, so we won't load the backup file next time */
			currentTime := time.Now().UTC().Local()
			_ = os.Chtimes(fullpath, currentTime, currentTime)
			cfg.WriteLCfg()
		}
		return true
	}
	return false
}

func handleDesync(input *handleData) bool {
	/******************
	 * CAPTURE DESYNC
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Info") {
		if strings.Contains(input.noTimecode, "New RCON connection from") {
			cwlog.DoLogCW(input.noTimecode)
			return true
		}
		if strings.Contains(input.noTimecode, "DesyncedWaitingForMap") {
			cwlog.DoLogGame(input.noTimecode)
			cwlog.DoLogCW("desync: " + input.noTimecode)
			return true
		}
	}
	return false
}

func handleCrashes(input *handleData) bool {

	/* *****************
	 * CAPTURE CRASHES
	 ******************/
	if strings.HasPrefix(input.line, "__level__/") {
		cwlog.DoLogCW(input.line)
	}
	if strings.HasPrefix(input.noTimecode, "Error") {
		cwlog.DoLogCW(input.noTimecode)

		/* Lock error */
		if strings.Contains(input.noTimecode, "Couldn't acquire exclusive lock") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio is already running.")
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Mod Errors */
		if strings.Contains(input.noTimecode, "caused a non-recoverable error.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua error and will reboot.**")
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Stack traces */
		if strings.Contains(input.noTimecode, "Hosting multiplayer game failed") {
			if strings.Contains(input.noTimecode, "directory iterator cannot open directory") {
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio didn't find any save-games.")
			} else {
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio was unable to load a multiplayer game.")
			}
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* level.dat */
		if strings.Contains(input.noTimecode, "level.dat not found.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Stack traces */
		if strings.Contains(input.noTimecode, "Unexpected error occurred.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Factorio crashed.**")
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		if strings.Contains(input.noTimecode, "CommandLineMultiplayer") {
			if strings.Contains(input.noTimecode, "No latest save file found in") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "No save-game found.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
		}
		if strings.Contains(input.noTimecode, "Scenario") && strings.HasSuffix(input.noTimecode, "not found") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Invalid scenario specified, clearing scenario setting.")
			cfg.Local.Settings.Scenario = ""
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}

		/* Multiplayer manger */
		if strings.HasPrefix(input.noTimecode, "Error ServerMultiplayerManager") {
			fact.CMS(cfg.Global.Discord.ReportChannel, cfg.Global.GroupName+"-"+cfg.Local.Callsign+": "+cfg.Local.Name+":\n"+input.noTimecode)
		}
		if strings.Contains(input.noTimecode, "MultiplayerManager failed:") {

			if strings.Contains(input.noTimecode, "cannot be loaded because it is higher than the game version") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio version is too old for the save game.**")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			if strings.Contains(input.noTimecode, "syntax error") || strings.Contains(input.noTimecode, "unexpected symbol") ||
				strings.Contains(input.noTimecode, "expected") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua syntax error and will stop.**")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			if strings.Contains(input.noTimecode, "Error while running command") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua command error.**")
				return true
			}
			if strings.Contains(input.noTimecode, "info.json not found") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			/* Bad zip file */
			if strings.Contains(input.noTimecode, "(Bad zip file)") {
				if input.trimmedWordsLen > 6 {
					if strings.HasSuffix(input.trimmedWords[7], ".zip") || strings.HasSuffix(input.trimmedWords[7], ".tmp.zip") {
						err := os.Remove(input.trimmedWords[7])
						if err != nil {
							cwlog.DoLogCW("Unable to remove bad zip file: " + input.trimmedWords[7])
							fact.FactAutoStart = false
						} else {
							cwlog.DoLogCW("Removed bad zip file: " + input.trimmedWords[7])
						}
						return true
					}
				}
			}
			/* Corrupt savegame */
			if strings.Contains(input.noTimecode, "Closing file") {
				errs := os.Remove(fact.GameMapPath)

				if errs != nil {
					cwlog.DoLogCW("Unable to delete corrupt savegame. Details:\nfile: %v\nerr: %v", fact.GameMapPath, errs)
					fact.FactAutoStart = false
					fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to remove corrupted save-game.")
				} else {
					cwlog.DoLogCW("Deleted corrupted savegame.")
					fact.CMS(cfg.Local.Channel.ChatChannel, "Save-game corrupted, performing automatic roll-back.")
				}

				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}

			if strings.Contains(input.noTimecode, "Exception at tick") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio crashed.**")
				return true
			}
		}
		return true
	}
	return false
}

func handleChatMsg(input *handleData) bool {
	/************************
	 * FACTORIO CHAT MESSAGES
	 ************************/
	if strings.HasPrefix(input.noDatestamp, "[CHAT]") || strings.HasPrefix(input.noDatestamp, "[SHOUT]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			input.noDatestampList[1] = strings.Replace(input.noDatestampList[1], ":", "", -1)
			pname := input.noDatestampList[1]

			if pname != "<server>" {

				var nores int = glob.NoResponseCount

				glob.ChatterLock.Lock()

				//Do not ban for chat spam if game is lagging
				if nores < 5 && fact.PlayerLevelGet(pname, true) != 255 {
					var bbuf string

					//Automatically ban people for chat spam
					if time.Since(glob.ChatterList[pname]) < constants.SpamSlowThres {
						glob.ChatterSpamScore[pname]++
						glob.ChatterList[pname] = time.Now()
					} else if time.Since(glob.ChatterList[pname]) < constants.SpamFastThres {
						glob.ChatterSpamScore[pname] += 2
						glob.ChatterList[pname] = time.Now()
					} else if time.Since(glob.ChatterList[pname]) > constants.SpamCoolThres {
						if glob.ChatterSpamScore[pname] > 0 {
							glob.ChatterSpamScore[pname]--
						}
						glob.ChatterList[pname] = time.Now()
					} else if time.Since(glob.ChatterList[pname]) > constants.SpamResetThres {
						glob.ChatterSpamScore[pname] = 0
						glob.ChatterList[pname] = time.Now()
					}

					if glob.ChatterSpamScore[pname] > constants.SpamScoreWarning {
						if !cfg.Global.Options.DisableSpamProtect {
							bbuf = fmt.Sprintf("/whisper %v [color=red]*** SPAMMING / FLOODING WARNING! (slow down) ***[/color]\n", pname)
							fact.WriteFact(bbuf)
						}
					} else if glob.ChatterSpamScore[pname] > constants.SpamScoreLimit {
						if !cfg.Global.Options.DisableSpamProtect {
							if cfg.Global.Paths.URLs.LogPath != "" {
								bbuf = fmt.Sprintf("/ban %v Spamming / flooding (auto-ban) %v", pname, cfg.GetGameLogURL())
							} else {
								bbuf = fmt.Sprintf("/ban %v Spamming / flooding (auto-ban)", pname)
							}

							glob.PlayerListLock.Lock()
							if glob.PlayerList[pname] != nil &&
								!glob.PlayerList[pname].AlreadyBanned {
								glob.PlayerList[pname].AlreadyBanned = true
								fact.WriteFact(bbuf)
							}
							fact.WriteFact("/purge " + pname)
							glob.PlayerListLock.Unlock()

						}
						glob.ChatterSpamScore[pname] = 0
					}
				} else {
					/* Lower score if server isn't responding */
					if glob.ChatterSpamScore[pname] > 0 {
						glob.ChatterSpamScore[pname]--
					}
				}

				glob.ChatterLock.Unlock()

				cmess := strings.Join(input.noDatestampList[2:], " ")
				cmess = sclean.UnicodeCleanup(cmess)
				cmess = sclean.EscapeDiscordMarkdown(cmess)
				cmess = sclean.RemoveFactorioTags(cmess)

				if len(cmess) > 500 {
					cmess = fmt.Sprintf("%s**(message cut, too long!)**", sclean.TruncateStringEllipsis(cmess, 500))
				}

				if cmess == "" {
					return true
				}

				/* Yeah, on different thread please. */
				go func(ptemp string) {
					fact.UpdateSeen(ptemp)
				}(pname)

				did := disc.GetDiscordIDFromFactorioName(pname)
				dname := disc.GetNameFromID(did)
				avatar := disc.GetDiscordAvatarFromId(did, 64)
				factname := sclean.UnicodeCleanup(pname)
				factname = sclean.TruncateString(factname, 25)

				fbuf := ""
				/* Filter Factorio names */

				factname = sclean.UnicodeCleanup(factname)
				factname = sclean.EscapeDiscordMarkdown(factname)
				if dname != "" {
					fbuf = fmt.Sprintf("`%v` **%s**: %s", fact.Gametime, factname, cmess)
				} else {
					fbuf = fmt.Sprintf("`%v` %s: %s", fact.Gametime, factname, cmess)
				}

				/* Remove all but letters */
				filter, _ := regexp.Compile("[^a-zA-Z]+")

				/* Name to lowercase */
				dnamelower := strings.ToLower(dname)
				fnamelower := strings.ToLower(pname)

				/* Reduce to letters only */
				dnamereduced := filter.ReplaceAllString(dnamelower, "")
				fnamereduced := filter.ReplaceAllString(fnamelower, "")

				/* If we find Discord name, and Discord name and Factorio name don't contain the same name */
				if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {
					/* Slap data into embed format. */
					myembed := embed.NewEmbed().
						SetAuthor("@"+dname, avatar).
						SetDescription(fbuf).
						MessageEmbed

						/* Send it off! */
					err := disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, myembed)
					if err != nil {
						/* On failure, send normal message */
						cwlog.DoLogCW("Failed to send chat embed.")
					} else {
						/* Stop if succeeds */
						return true
					}
				}
				fact.CMS(cfg.Local.Channel.ChatChannel, fbuf)
			}
			return true
		}
		return true
	}
	return false
}

func handleCmdMsg(input *handleData) bool {
	/******************
	 * COMMAND REPORTING
	 ******************/
	if strings.HasPrefix(input.line, "[CMD]") && !strings.Contains(input.line, "/cchat") {
		cwlog.DoLogGame(input.line)
		return true
	}
	return false
}

func handleOnlineMsg(input *handleData) bool {
	/* ****************
	 * "/online"
	 * This is specific to our soft-mod
	 ******************/
	newMode := false
	if strings.HasPrefix(input.line, "[ONLINE]") || strings.HasPrefix(input.line, "[ONLINE2]") {
		tag := "[ONLINE] "
		if strings.HasPrefix(input.line, "[ONLINE2]") {
			tag = "[ONLINE2] "
			newMode = true
		}
		newPlayerList := []glob.OnlinePlayerData{}
		count := 0

		//cwlog.DoLogCW(input.line)
		line := strings.TrimPrefix(input.line, tag)

		players := strings.Split(line, ";")
		if len(players) > 0 {
			for _, p := range players {
				fields := strings.Split(p, ",")
				if len(fields) > 3 {

					//name,score,time,type;
					pname := fields[0]
					pscore := fields[1]
					ptime := fields[2]
					ptype := fields[3]
					pafk := fields[4]

					plevel := fact.StringToLevel(ptype)

					if pname != "" {
						/* Mark as seen, async */
						go fact.UpdateSeen(pname)

						/* Check if user is banned */
						banlist.CheckBanList(pname)

						timeInt, _ := strconv.Atoi(ptime)
						scoreInt, _ := strconv.Atoi(pscore)
						/* Handle new compacted format */
						if newMode {
							timeInt = (timeInt * 60 * 60)
							scoreInt = (scoreInt * 60 * 60)
						}
						newPlayerList = append(newPlayerList, glob.OnlinePlayerData{Name: pname, ScoreTicks: scoreInt, TimeTicks: timeInt, Level: plevel, AFK: pafk})
						count++
					}

				}
			}
			if count > 0 {
				fact.NumPlayers = (count)
				fact.OnlinePlayersLock.Lock()
				glob.OnlinePlayers = newPlayerList

				fact.OnlinePlayersLock.Unlock()
				return true
			}
		}

		/* Otherwise clear list */
		fact.NumPlayers = (0)
		glob.OnlinePlayers = []glob.OnlinePlayerData{}

		return true
	}
	return false
}
