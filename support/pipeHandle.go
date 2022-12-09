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
func handleIdiots(line string) bool {
	/* Protect players from dumb mistakes with registration codes */
	if ProtectIdiots(line) {
		buf := "You didn't enter that as a command. You posted your registration code publicly, invalidating code. Please read the directions more carefully."
		fact.FactChat(buf)
		fact.CMS(cfg.Local.Channel.ChatChannel, buf)
		return true
	}

	return false
}

func handleGameTime(lowerCaseLine string, lowerCaseList []string, lowerCaseListlen int) {
	/********************
	 * GET FACTORIO TIME
	 ********************/
	if strings.Contains(lowerCaseLine, " second") || strings.Contains(lowerCaseLine, " minute") || strings.Contains(lowerCaseLine, " hour") || strings.Contains(lowerCaseLine, " day") {

		day := 0
		hour := 0
		minute := 0
		second := 0

		if lowerCaseListlen > 1 {

			for x := 0; x < lowerCaseListlen; x++ {
				if strings.Contains(lowerCaseList[x], "day") {
					day, _ = strconv.Atoi(lowerCaseList[x-1])
				} else if strings.Contains(lowerCaseList[x], "hour") {
					hour, _ = strconv.Atoi(lowerCaseList[x-1])
				} else if strings.Contains(lowerCaseList[x], "minute") {
					minute, _ = strconv.Atoi(lowerCaseList[x-1])
				} else if strings.Contains(lowerCaseList[x], "second") {
					second, _ = strconv.Atoi(lowerCaseList[x-1])
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
			fact.GametimeString = lowerCaseLine
			fact.Gametime = newtime
		}
		/* This might block stuff by accident, don't do it */
		//continue
	}
}

func handlePlayerReport(line string, lineList []string, lineListlen int) bool {
	/******************
	 * Player REPORT
	 ******************/
	if strings.HasPrefix(line, "[REPORT]") {
		cwlog.DoLogGame(line)
		if lineListlen >= 3 {
			buf := fmt.Sprintf("**PLAYER REPORT:**\nServer: %v, Reporter: %v: Report:\n %v",
				cfg.Local.Callsign+"-"+cfg.Local.Name, lineList[1], strings.Join(lineList[2:], " "))
			fact.CMS(cfg.Global.Discord.ReportChannel, buf)
		}
		return true
	}

	return false
}

func handlePlayerRegister(line string, lineList []string, lineListlen int) bool {
	/******************
	 * ACCESS
	 ******************/
	if strings.HasPrefix(line, "[ACCESS]") {
		if lineListlen >= 4 {
			/* Format:
			 * print("[ACCESS] " .. ptype .. " " .. player.name .. " " .. param.parameter) */

			ptype := lineList[1]
			pname := lineList[2]
			code := strings.Join(lineList[3:lineListlen], "")

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
					if strings.EqualFold(ptype, "trusted") {
						newrole = cfg.Global.Discord.Roles.Member
						plevel = 1
					} else if strings.EqualFold(ptype, "regular") {
						newrole = cfg.Global.Discord.Roles.Regular
						plevel = 2
					} else if strings.EqualFold(ptype, "moderator") {
						newrole = cfg.Global.Discord.Roles.Moderator
						plevel = 255
					} else {
						newrole = cfg.Global.Discord.Roles.New
						plevel = 0
					}

					discid := disc.GetDiscordIDFromFactorioName(pname)
					factname := disc.GetFactorioNameFromDiscordID(pid)

					if strings.EqualFold(discid, pid) && strings.EqualFold(factname, pname) {
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This Factorio account, and Discord account are already connected! Setting role, if needed.", pname))
						codegood = true
						/* Do not break, process */
					} else if discid != "" {
						cwlog.DoLogCW(fmt.Sprintf("Factorio player '%s' tried to connect a Discord account, that is already connected to a different Factorio account.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] That Discord name is already connected to a different Factorio account.", pname))
						codegood = false
						continue
					} else if factname != "" {
						cwlog.DoLogCW(fmt.Sprintf("Factorio player '%s' tried to connect their Factorio account, that is already connected to a different Discord account.", pname))
						fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] This Factorio account is already connected to a different Discord account.", pname))
						codegood = false
						continue
					}

					if codegood {
						fact.PlayerSetID(pname, pid, plevel)

						guild := disc.Guild
						if guild != nil && disc.DS != nil {
							errrole, regrole := disc.RoleExists(guild, newrole)

							if !errrole {
								fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("Sorry, there is an error. I could not find the Discord role '%s'.", newrole))
								fact.WriteFact(fmt.Sprintf("/cwhisper %s  [SYSTEM] Sorry, there was an internal error, I could not find the Discord role '%s' Let the moderators know!", newrole, pname))
								continue
							}

							erradd := disc.SmartRoleAdd(cfg.Global.Discord.Guild, pid, regrole.ID)

							if erradd != nil || disc.DS == nil {
								fact.CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("Sorry, there is an error. I could not assign the Discord role '%s'.", newrole))
								fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, there was an error, could not assign role '%s' Let the moderators know!", newrole, pname))
								continue
							}
							fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Registration complete!", pname))
							fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Registered player: %v", pname))
							continue
						} else {
							cwlog.DoLogCW("No guild info.")
							fact.CMS(cfg.Local.Channel.ChatChannel, "Sorry, I couldn't find the guild info!")
							continue
						}
					}
					continue
				}
			} /* End of loop */
			glob.PasswordListLock.Unlock()
			if !codefound {
				cwlog.DoLogCW(fmt.Sprintf("Factorio player '%s', tried to use an invalid or expired code.", pname))
				fact.WriteFact(fmt.Sprintf("/cwhisper %s [SYSTEM] Sorry, that code is invalid or expired. Make sure you are entering the code on the correct Factorio server!", pname))
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

func handleOnlinePlayers(line string, lineList []string, lineListlen int) bool {
	/* ***********************
	 * CAPTURE ONLINE PLAYERS
	 ************************/
	if strings.HasPrefix(line, "Online players") {

		if lineListlen > 2 {
			poc := strings.Join(lineList[2:], " ")
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

func handlePlayerJoin(NoDS string, NoDSlist []string, NoDSlistlen int) bool {
	/******************
	 * JOIN AREA
	 *****************/
	if strings.HasPrefix(NoDS, "[JOIN]") {
		cwlog.DoLogGame(NoDS)

		if NoDSlistlen > 1 {
			pname := sclean.StripControlAndSubSpecial(NoDSlist[1])
			banlist.CheckBanList(pname)
			plevelname := fact.AutoPromote(pname)

			pname = sclean.EscapeDiscordMarkdown(pname)

			buf := fmt.Sprintf("`%v` **%s joined**%s", fact.Gametime, pname, plevelname)
			fact.CMS(cfg.Local.Channel.ChatChannel, buf)

			if glob.SoftModVersion != constants.Unknown &&
				fact.FactIsRunning &&
				fact.FactorioBooted {
				fact.UpdateScheduleDesc()
				fact.WriteFact("/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")")
			}

			/* Give people patreon/nitro tags in-game. */
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
		return true
	}
	return false
}

func handlePlayerLeave(NoDS string, line string, NoDSlist []string, NoDSlistlen int) bool {
	/******************
	 * LEAVE
	 ******************/
	if strings.HasPrefix(NoDS, "[LEAVE]") &&
		/* Suppress quit messages from map load */
		!fact.FactorioBootedAt.IsZero() &&
		time.Since(fact.FactorioBootedAt) > time.Second*30 {

		cwlog.DoLogGame(NoDS)

		/* Handle softmod and vanilla */
		if glob.SoftModVersion == constants.Unknown {
			if NoDSlistlen > 1 {
				buf := strings.Join(NoDSlist[1:NoDSlistlen], " ")
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
			}
		}

		if NoDSlistlen > 1 {
			pname := NoDSlist[1]

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(pname)
		}
		return true
	}
	return false
}

func handleActMsg(line string, lineList []string, lineListLen int) bool {
	/******************
	 * ACT AREA
	 ******************/

	if strings.HasPrefix(line, "[ACT]") {

		cwlog.DoLogGame(line)
		if lineListLen > 2 {

			pname := lineList[1]
			action := lineList[2]

			words := lineList[3:]
			numWords := len(words) - 1

			if pname == "" {
				return true
			}

			go fact.UpdateSeen(pname)
			if pname != "" && glob.PlayerSus != nil {
				p := disc.GetPlayerDataFromName(pname)
				if p != nil && p.Name != "" && p.Level < 2 {

					glob.PlayerSusLock.Lock()

					if strings.Contains(action, "placed-ghost") {
						glob.PlayerSus[pname] -= 2
					} else if strings.Contains(action, "mined-ghost") {
						glob.PlayerSus[pname] -= 1
					} else if strings.Contains(action, "placed") {
						glob.PlayerSus[pname]--
					} else if strings.Contains(action, "mined") {
						glob.PlayerSus[pname]++
					} else if strings.Contains(action, "deconstructing") {
						if numWords > 3 {
							areaString := words[4]
							areaClean := strings.ReplaceAll(areaString, "sq", "")
							area, _ := strconv.Atoi(areaClean)
							glob.PlayerSus[pname] += (area / 200)
						}
					}

					thresh := constants.SusWarningThresh
					if p.Level > 0 {
						thresh += 1000
					}
					if glob.PlayerSus[pname] > thresh {

						if time.Since(glob.LastSusWarning) > time.Minute*2 {
							glob.LastSusWarning = time.Now()

							if !cfg.Global.Options.ShutupSusWarn {
								sbuf := fmt.Sprintf("*WARNING*: Player: '%v': Possible suspicious activity!)", pname)
								fact.FactChat("[color=red]" + sbuf + "[/color]")
								fact.CMS(cfg.Local.Channel.ChatChannel, sbuf)
								sbuf = cfg.Global.GroupName + "-" + cfg.Local.Callsign + ": " + cfg.Local.Name + ": " + sbuf
								fact.CMS(cfg.Global.Discord.ReportChannel, sbuf)
							}

							glob.PlayerSus[pname] = 0
						}
					}

					glob.PlayerSusLock.Unlock()
				}
			}
		}
		return true
	}

	return false
}

func handleSoftModMsg(line string, lineList []string, lineListlen int) bool {
	/******************
	 * MSG AREA
	 ******************/
	if strings.HasPrefix(line, "[MSG]") {
		cwlog.DoLogGame(line)

		if lineListlen > 0 {
			ctext := strings.Join(lineList[1:], " ")

			/* Clean strings */
			cmess := sclean.StripControlAndSubSpecial(ctext)
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

		if lineListlen > 1 {
			trustname := lineList[1]

			if trustname != "" {

				if strings.Contains(line, " is now a member!") {
					fact.PlayerLevelSet(trustname, 1, false)
					fact.AutoPromote(trustname)
					return true
				} else if strings.Contains(line, " is now a regular!") {
					fact.PlayerLevelSet(trustname, 2, false)
					fact.AutoPromote(trustname)
					return true
				} else if strings.Contains(line, " moved to moderators group") {
					fact.PlayerLevelSet(trustname, 255, false)
					fact.AutoPromote(trustname)
					return true
				} else if strings.Contains(line, " has nil permissions.") {
					fact.AutoPromote(trustname)
					return true
				}
			}
		}
		return true
	}
	return false

}

func handleSlowConnect(NoTC string, line string) {
	/* *****************
	 * Slow on catch-up
	 ******************/
	if cfg.Local.Options.SoftModOptions.SlowConnect.Enabled {

		if strings.HasPrefix(NoTC, "Info ServerMultiplayerManager") {

			if strings.Contains(line, "removing peer") {

				/* We do this, so we can get a corrected player count */
				if glob.SoftModVersion != constants.Unknown {
					fact.WriteFact(glob.OnlineCommand)
				}

			} else if strings.Contains(line, "oldState(ConnectedLoadingMap) newState(TryingToCatchUp)") {
				SlowConnectStart()
			} else if strings.Contains(line, "oldState(WaitingForCommandToStartSendingTickClosures) newState(InGame)") {
				SlowConnectEnd()
			} else if strings.Contains(line, "oldState(TryingToCatchUp) newState(DisconnectScheduled") {
				SlowConnectEnd()
			}

		}
	}
}

func SlowConnectStart() {
	if cfg.Local.Options.SoftModOptions.SlowConnect.ConnectSpeed <= 0.0 {
		fact.WriteFact("/gspeed 0.5")
	} else {
		fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.Options.SoftModOptions.SlowConnect.ConnectSpeed))
	}

	tn := time.Now()
	fact.SlowConnectLock.Lock()
	fact.SlowConnectTimer = tn.Unix()
	fact.SlowConnectEvents++
	fact.SlowConnectLock.Unlock()
}

func SlowConnectEnd() {

	fact.SlowConnectLock.Lock()

	fact.SlowConnectEvents--
	if fact.SlowConnectEvents <= 0 {
		fact.SlowConnectEvents = 0
		fact.SlowConnectTimer = 0

		if cfg.Local.Options.SoftModOptions.SlowConnect.Speed >= 0.0 {
			fact.WriteFact("/gspeed " + fmt.Sprintf("%v", cfg.Local.Options.SoftModOptions.SlowConnect.Speed))
		} else {
			fact.WriteFact("/gspeed 1.0")
		}
	}

	fact.SlowConnectLock.Unlock()
}

func handleMapLoad(NoTC string, NoDSlist []string, NoTClist []string, NoTClistlen int) bool {
	/******************
	 * MAP LOAD
	 ******************/
	if strings.HasPrefix(NoTC, "Loading map") {
		cwlog.DoLogGame(NoTC)

		/* Strip file path */
		if NoTClistlen > 3 {
			fullpath := NoTClist[2]
			size := NoTClist[3]
			sizei, _ := strconv.Atoi(size)
			fullpath = strings.Replace(fullpath, ":", "", -1)

			regaa := regexp.MustCompile(`\/.*?\/saves\/`)
			filename := regaa.ReplaceAllString(fullpath, "")

			fact.GameMapName = filename
			fact.GameMapPath = fullpath
			fact.LastSaveName = filename

			buf := fmt.Sprintf("Loading map %s (%v)...", filename, humanize.Bytes(uint64(sizei)))
			cwlog.DoLogCW(buf)
		} else { /* Just in case */
			cwlog.DoLogCW("Loading map...")
		}
		return true
	}
	return false
}

func handleModLoad(NoTC string) bool {
	/******************
	 * LOADING MOD
	 ******************/
	if strings.HasPrefix(NoTC, "Loading mod") {

		if !strings.Contains(NoTC, "base") &&
			!strings.Contains(NoTC, "core") &&
			!strings.Contains(NoTC, "settings") {

			parts := strings.Split(NoTC, " ")
			numParts := len(parts) - 1
			if numParts >= 4 {

				modName := strings.Join(parts[2:numParts-1], " ")
				modName = sclean.AlphaOnly(modName)

				found := false
				for _, m := range fact.ModList {
					if strings.EqualFold(m, modName) {
						found = true
					}
				}
				if !found {
					fact.ModList = append(fact.ModList, modName)
					cwlog.DoLogGame(NoTC)
				}
			}
		}
	}
	/* Dont eat, factorio version handle uses this too */
	return false
}

func handleBan(NoDS string, NoDSlist []string, NoDSlistlen int) bool {
	/******************
	 * BAN
	 ******************/
	if strings.HasPrefix(NoDS, "[BAN]") {

		glob.PlayerListWriteLock.Lock()
		defer glob.PlayerListWriteLock.Unlock()

		cwlog.DoLogGame(NoDS)

		if NoDSlistlen > 1 {
			trustname := NoDSlist[1]

			if strings.Contains(NoDS, "was banned by") {

				if strings.Contains(NoDS, "Reason") {

					reasonList := strings.Split(NoDS, "Reason: ")

					//Report bans
					if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) {
						buf := fmt.Sprintf("M45 ban: %v, Reason: %v", trustname, reasonList[1])
						fact.CMS(cfg.Global.Discord.ReportChannel, buf)
					}

					fact.PlayerSetBanReason(trustname, reasonList[1], false)
				} else {
					if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) {
						buf := fmt.Sprintf("M45 ban: %v", trustname)
						fact.CMS(cfg.Global.Discord.ReportChannel, buf)
					}
					fact.PlayerLevelSet(trustname, -1, false)
				}
			}

			fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.Gametime, strings.Join(NoDSlist[1:], " ")))
		}
		return true
	}
	return false
}

func handleSVersion(line string, lineList []string, lineListlen int) bool {
	/******************
	 * SVERSION
	 ******************/
	if strings.HasPrefix(line, "[SVERSION]") {
		cwlog.DoLogGame(line)

		if lineListlen > 0 {
			glob.SoftModVersion = lineList[1]
			glob.OnlineCommand = constants.SoftModOnlineCMD
			cwlog.DoLogCW("Softmod detected: " + glob.SoftModVersion)
		}
		ConfigSoftMod()
		return true
	}
	return false
}

func handleUnBan(NoDS string, NoDSlist []string, NoDSlistlen int) bool {
	/******************
	 * UNBAN
	 ******************/
	if strings.HasPrefix(NoDS, "[UNBANNED]") {
		cwlog.DoLogGame(NoDS)

		if NoDSlistlen > 1 {
			trustname := NoDSlist[1]

			if strings.Contains(NoDS, "was unbanned by") {
				if fact.PlayerLevelGet(trustname, true) < 0 {
					fact.PlayerLevelSet(trustname, 0, false)
				}
			}

			fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.Gametime, strings.Join(NoDSlist[1:], " ")))
		}
		return true
	}
	return false
}

func handleFactGoodbye(NoTC string) bool {
	/******************
	 * GOODBYE
	 ******************/
	if strings.HasPrefix(NoTC, "Goodbye") {
		cwlog.DoLogGame(NoTC)

		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio is now offline.")
		fact.FactorioBooted = false
		fact.FactorioBootedAt = time.Time{}
		fact.SetFactRunning(false)

		return true
	}
	return false
}

func handleFactReady(NoTC string) bool {
	/*****************
	 * READY MESSAGE
	 ******************/
	if strings.HasPrefix(NoTC, "Info RemoteCommandProcessor") && strings.Contains(NoTC, "Starting RCON interface") {

		fact.FactorioBooted = true
		fact.FactorioBootedAt = time.Now()
		fact.SetFactRunning(true)
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio "+fact.FactorioVersion+" is now online.")
		fact.WriteFact("/sversion")
	}
	return false
}

func handleIncomingAnnounce(NoTC string, words []string, numwords int) bool {
	/********************************
	 * Announce incoming connections
	 ********************************/
	if strings.Contains(NoTC, "Queuing ban recommendation check for user ") {
		if numwords > 1 {
			pName := words[numwords-1]

			dmsg := fmt.Sprintf("`%v` %v is connecting.", fact.Gametime, pName)
			fmsg := fmt.Sprintf("%v is connecting.", pName)
			fact.FactChat(fmsg)
			fact.CMS(cfg.Local.Channel.ChatChannel, dmsg)
			return true
		}
	}
	return false
}

func handleFactVersion(NoTC string, line string, NoTClist []string, NoTClistlen int) {
	/* **********************
	 * GET FACTORIO VERSION
	 ***********************/
	if strings.HasPrefix(NoTC, "Loading mod base") {
		cwlog.DoLogGame(NoTC)
		if NoTClistlen > 3 {
			fact.FactorioVersion = NoTClist[3]

			fv := strings.Split(fact.FactorioVersion, ".")
			fvl := len(fv)

			var a, b, c int

			if fvl == 3 {
				a, _ = strconv.Atoi(fv[0])
				b, _ = strconv.Atoi(fv[1])
				c, _ = strconv.Atoi(fv[2])
			}

			fact.FactorioVersionA = a
			fact.FactorioVersionB = b
			fact.FactorioVersionC = c
		}
	}

}

func handleSaveMsg(NoTC string) bool {
	/*************************
	 * CAPTURE SAVE MESSAGES
	 *************************/
	if strings.HasPrefix(NoTC, "Info AppManager") && strings.Contains(NoTC, "Saving to") {
		if !cfg.Local.Options.HideAutosaves {
			savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
			savmatch := savreg.FindStringSubmatch(NoTC)
			if len(savmatch) > 1 {
				if !cfg.Local.Options.HideAutosaves {
					buf := fmt.Sprintf("`%v` 💾 %s", fact.Gametime, savmatch[1])
					fact.LogCMS(cfg.Local.Channel.ChatChannel, buf)
				}
				fact.LastSaveName = savmatch[1]
			}
		}
		return true
	}
	return false
}

func handleExitSave(NoTC string, NoTClist []string, NoTClistlen int) bool {
	/*****************************
	 * CAPTURE MAP NAME, ON EXIT
	 *****************************/
	if strings.HasPrefix(NoTC, "Info MainLoop") && strings.Contains(NoTC, "Saving map as") {
		cwlog.DoLogGame(NoTC)

		/* Strip file path */
		if NoTClistlen > 5 {
			fullpath := NoTClist[5]
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
			cwlog.DoLogCW(fmt.Sprintf("Map saved as: %v, backup: %v", filename, newName))
			fact.LastSaveName = filename

			/* Open the quit-save */
			from, erra := os.Open(fullpath)
			if erra != nil {

				buf := fmt.Sprintf("An error occurred when attempting to read the save to backup: %s", erra)
				cwlog.DoLogCW(buf)
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
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
				buf := fmt.Sprintf("An error occurred when attempting to write the backup save: %s", errc)
				cwlog.DoLogCW(buf)
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

func handleDesync(NoTC string, line string) bool {
	/******************
	 * CAPTURE DESYNC
	 ******************/
	if strings.HasPrefix(NoTC, "Info") {

		if strings.Contains(NoTC, "DesyncedWaitingForMap") {
			cwlog.DoLogGame(NoTC)
			cwlog.DoLogCW("desync: " + NoTC)
			return true
		}
	}
	return false
}

func handleCrashes(NoTC string, line string, words []string, numwords int) bool {

	/* *****************
	 * CAPTURE CRASHES
	 ******************/
	if strings.HasPrefix(NoTC, "Error") {
		cwlog.DoLogGame(NoTC)

		/* Lock error */
		if strings.Contains(NoTC, "Couldn't acquire exclusive lock") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio is already running.")
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Mod Errors */
		if strings.Contains(NoTC, "caused a non-recoverable error.") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio encountered a lua error and will reboot.")
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Stack traces */
		if strings.Contains(NoTC, "Hosting multiplayer game failed") {
			if strings.Contains(NoTC, "directory iterator cannot open directory") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio didn't find any save-games.")
			} else {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio was unable to load a multiplayer game.")
			}
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* level.dat */
		if strings.Contains(NoTC, "level.dat not found.") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
			fact.FactAutoStart = false
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		/* Stack traces */
		if strings.Contains(NoTC, "Unexpected error occurred.") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio crashed.")
			fact.FactorioBooted = false
			fact.SetFactRunning(false)
			return true
		}
		if strings.Contains(NoTC, "CommandLineMultiplayer") {
			if strings.Contains(NoTC, "No latest save file found in") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "No save-game found.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
		}
		/* Multiplayer manger */
		if strings.Contains(NoTC, "MultiplayerManager failed:") {
			if strings.Contains(NoTC, "cannot be loaded because it is higher than the game version") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio version is too old for the save game.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			if strings.Contains(NoTC, "syntax error") || strings.Contains(NoTC, "unexpected symbol") ||
				strings.Contains(NoTC, "expected") || strings.Contains(NoTC, ".lua:") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio encountered a lua syntax error and will stop.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			if strings.Contains(NoTC, "info.json not found") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
				fact.FactAutoStart = false
				fact.FactorioBooted = false
				fact.SetFactRunning(false)
				return true
			}
			/* Bad zip file */
			if strings.Contains(NoTC, "(Bad zip file)") {
				if numwords > 6 {
					if strings.HasSuffix(words[7], ".zip") || strings.HasSuffix(words[7], ".tmp.zip") {
						err := os.Remove(words[7])
						if err != nil {
							cwlog.DoLogCW("Unable to remove bad zip file: " + words[7])
							fact.FactAutoStart = false
						} else {
							cwlog.DoLogCW("Removed bad zip file: " + words[7])
						}
						return true
					}
				}
			}
			/* Corrupt savegame */
			if strings.Contains(NoTC, "Closing file") {
				errs := os.Remove(fact.GameMapPath)

				if errs != nil {
					cwlog.DoLogCW(fmt.Sprintf("Unable to delete corrupt savegame. Details:\nfile: %v\nerr: %v", fact.GameMapPath, errs))
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
		}
		return true
	}
	return false
}

func handleChatMsg(NoDS string, line string, NoDSlist []string, NoDSlistlen int) bool {
	/************************
	 * FACTORIO CHAT MESSAGES
	 ************************/
	if strings.HasPrefix(NoDS, "[CHAT]") || strings.HasPrefix(NoDS, "[SHOUT]") {
		cwlog.DoLogGame(NoDS)

		if NoDSlistlen > 1 {
			NoDSlist[1] = strings.Replace(NoDSlist[1], ":", "", -1)
			pname := NoDSlist[1]

			if pname != "<server>" {

				var nores int = glob.NoResponseCount

				glob.ChatterLock.Lock()

				//Do not ban for chat spam if game is lagging
				if nores < 5 && fact.PlayerLevelGet(pname, true) != 255 {
					var bbuf string

					//Automatically ban people for chat spam
					//TODO: Make this configurable
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
								newmapurl := fmt.Sprintf("https://%v%v%v%v",
									cfg.Global.Paths.URLs.Domain,
									cfg.Global.Paths.URLs.PathPrefix,
									cfg.Global.Paths.URLs.LogPath,
									strings.TrimPrefix(glob.GameLogName, "log/"))

								bbuf = fmt.Sprintf("/ban %v Spamming / flooding (auto-ban) %v", pname, newmapurl)
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

				cmess := strings.Join(NoDSlist[2:], " ")
				cmess = sclean.StripControlAndSubSpecial(cmess)
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
				dname := disc.GetNameFromID(did, false)
				avatar := disc.GetDiscordAvatarFromId(did, 64)
				factname := sclean.StripControlAndSubSpecial(pname)
				factname = sclean.TruncateString(factname, 25)

				fbuf := ""
				/* Filter Factorio names */

				factname = sclean.StripControlAndSubSpecial(factname)
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

func handleCmdMsg(line string) bool {
	/******************
	 * COMMAND REPORTING
	 ******************/
	if strings.HasPrefix(line, "[CMD]") {
		cwlog.DoLogGame(line)
		return true
	}
	return false
}

func handleOnlineMsg(line string) bool {
	/* ****************
	 * "/online"
	 ******************/
	if strings.HasPrefix(line, "[ONLINE]") {
		newPlayerList := []glob.OnlinePlayerData{}
		count := 0

		cwlog.DoLogGame(line)
		line = strings.TrimPrefix(line, "[ONLINE] ")

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
						fact.UpdateSeen(pname)

						banlist.CheckBanList(pname)

						timeInt, _ := strconv.Atoi(ptime)
						scoreInt, _ := strconv.Atoi(pscore)
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
