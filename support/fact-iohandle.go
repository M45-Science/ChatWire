package support

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	embed "github.com/Clinet/discordgo-embed"
)

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

			newtime := constants.Unknown
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
			if fact.ConnectPauseTimer == 0 && fact.PausedTicks <= 2 {
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

			/* Pause detection */
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
			fact.GametimeString = lowerCaseLine
			fact.Gametime = newtime

			fact.PausedTicksLock.Unlock()
			fact.GametimeLock.Unlock()
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
				cfg.Local.ServerCallsign+"-"+cfg.Local.Name, lineList[1], strings.Join(lineList[2:], " "))
			fact.CMS(cfg.Global.DiscordData.ReportChannelID, buf)
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
		if lineListlen == 4 {
			/* Format:
			 * print("[ACCESS] " .. ptype .. " " .. player.name .. " " .. param.parameter) */

			ptype := lineList[1]
			pname := lineList[2]
			code := lineList[3]

			/* Filter just in case, and so accidental spaces won't ruin passcodes */
			code = strings.ReplaceAll(code, " ", "")
			pname = strings.ReplaceAll(pname, " ", "")

			codegood := true
			codefound := false
			plevel := 0

			glob.PasswordListLock.Lock()
			for i, pass := range glob.PassList {
				if pass.Code == code {
					codefound = true
					/* Delete password from list */
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
							cwlog.DoLogCW("No guild info.")
							fact.CMS(cfg.Local.ChannelData.ChatID, "Sorry, I couldn't find the guild info!")
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
			fact.SetNumPlayers(nump)

			glob.RecordPlayersLock.Lock()
			if nump > glob.RecordPlayers {
				glob.RecordPlayers = nump

				/* New thread, avoid deadlock */
				go func() {
					fact.WriteRecord()
				}()

				buf := fmt.Sprintf("**New record!** Players online: %v", glob.RecordPlayers)
				fact.CMS(cfg.Local.ChannelData.ChatID, buf)

				/* write to Factorio as well */
				buf = fmt.Sprintf("New record! Players online: %v", glob.RecordPlayers)
				fact.WriteFact("/cchat " + buf)

			}
			glob.RecordPlayersLock.Unlock()

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
		fact.WriteFact("/p o c")

		if NoDSlistlen > 1 {
			pname := sclean.StripControlAndSubSpecial(NoDSlist[1])
			glob.NumLoginsLock.Lock()
			glob.NumLogins = glob.NumLogins + 1
			glob.NumLoginsLock.Unlock()
			plevelname := fact.AutoPromote(pname)

			pname = sclean.EscapeDiscordMarkdown(pname)

			buf := fmt.Sprintf("`%v` **%s joined**%s", fact.GetGameTime(), pname, plevelname)
			fact.CMS(cfg.Local.ChannelData.ChatID, buf)

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
	if strings.HasPrefix(NoDS, "[LEAVE]") {
		cwlog.DoLogGame(line)
		fact.WriteFact("/p o c")

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

			fact.CMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%v` **%s**", fact.GetGameTime(), cmess))
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
				} else if strings.Contains(line, " moved to Admins group.") {
					fact.PlayerLevelSet(trustname, 255, false)
					fact.AutoPromote(trustname)
					return true
				} else if strings.Contains(line, " to the map!") && strings.Contains(line, "Welcome ") {
					btrustname := lineList[2]
					fact.AutoPromote(btrustname)
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
	if cfg.Local.SlowConnect.SlowConnect {

		tn := time.Now()

		if strings.HasPrefix(NoTC, "Info ServerMultiplayerManager") {

			if strings.Contains(line, "removing peer") {
				fact.WriteFact("/p o c")

				/* Fix for players leaving with no leave message */
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
	if strings.HasPrefix(NoTC, "Loading mod") && strings.HasSuffix(NoTC, "(data.lua)") {

		if !strings.Contains(NoTC, "base") && !strings.Contains(NoTC, "core") {
			cwlog.DoLogGame(NoTC)

			modName := strings.TrimPrefix(NoTC, "Loading mod ")
			modName = strings.TrimSuffix(modName, " (data.lua)")
			modName = strings.ReplaceAll(modName, " ", "-")
			fact.AddModLoadString(modName)
		}
		return true
	}
	return false
}

func handleBan(NoDS string, NoDSlist []string, NoDSlistlen int) bool {
	/******************
	 * BAN
	 ******************/
	if strings.HasPrefix(NoDS, "[BAN]") {
		cwlog.DoLogGame(NoDS)

		if NoDSlistlen > 1 {
			trustname := NoDSlist[1]

			if strings.Contains(NoDS, "was banned by") {
				fact.PlayerLevelSet(trustname, -1, false)
			}

			fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%v` %s", fact.GetGameTime(), strings.Join(NoDSlist[1:], " ")))
		}
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
				fact.PlayerLevelSet(trustname, 0, false)
			}

			fact.LogCMS(cfg.Local.ChannelData.ChatID, fmt.Sprintf("`%v` %s", fact.GetGameTime(), strings.Join(NoDSlist[1:], " ")))
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

		fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio is now offline.")
		fact.SetFactorioBooted(false)
		fact.SetFactRunning(false, false)
		return true
	}
	return false
}

func handleFactReady(NoTC string) bool {
	/*****************
	 * READY MESSAGE
	 ******************/
	if strings.HasPrefix(NoTC, "Info RemoteCommandProcessor") && strings.Contains(NoTC, "Starting RCON interface") {

		fact.SetFactorioBooted(true)
		fact.SetFactRunning(true, false)
		fact.LogCMS(cfg.Local.ChannelData.ChatID, "Factorio "+fact.FactorioVersion+" is now online.")
		fact.WriteFact("/p o c")

		fact.WriteFact("/cname " + strings.ToUpper(cfg.Local.ServerCallsign+"-"+cfg.Local.Name))

		/* Config new-player restrictions */
		if cfg.Local.SoftModOptions.RestrictMode {
			fact.WriteFact("/restrict on")
		} else {
			fact.WriteFact("/restrict off")
		}

		/* Config friendly fire */
		if cfg.Local.SoftModOptions.FriendlyFire {
			fact.WriteFact("/friendlyfire on")
		} else {
			fact.WriteFact("/friendlyfire off")
		}

		/* Config reset-interval */
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

		/* Patreon list */
		disc.RoleListLock.Lock()
		if len(disc.RoleList.Patreons) > 0 {
			fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ","))
		}
		if len(disc.RoleList.NitroBooster) > 0 {
			fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ","))
		}
		disc.RoleListLock.Unlock()
		return true
	}
	return false
}

func handleFixLockers(NoTC string) bool {
	/**********************
	 * FIX LOCKERS
	 *********************/
	if strings.Contains(NoTC, "ServerMultiplayerManager") {
		if strings.HasSuffix(NoTC, "changing state from(InGameSavingMap) to(InGame)") {

			fact.LockerLock.Lock()
			fact.LockerDetectStart = time.Now()
			fact.LockerStart = true
			fact.LockerLock.Unlock()
			return true

		} else if strings.HasSuffix(NoTC, "oldState(ConnectedWaitingForMap) newState(ConnectedDownloadingMap)") ||
			strings.Contains(NoTC, "Disconnect notification for peer") {

			fact.LockerLock.Lock()
			fact.LockerDetectStart = time.Now()
			fact.LockerStart = false
			fact.LockerLock.Unlock()
			return true
		}
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
			fact.LockerLock.Lock()
			fact.LastLockerName = pName
			fact.LockerLock.Unlock()

			dmsg := fmt.Sprintf("`%v` %v is connecting.", fact.GetGameTime(), pName)
			fmsg := fmt.Sprintf("%v is connecting.", pName)
			fact.WriteFact("/cchat " + fmsg)
			fact.CMS(cfg.Local.ChannelData.ChatID, dmsg)
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
		cwlog.DoLogGame(line)
		if NoTClistlen > 3 {
			fact.FactorioVersion = NoTClist[3]
		}
	}

}

func handleSaveMsg(NoTC string) bool {
	/*************************
	 * CAPTURE SAVE MESSAGES
	 *************************/
	if strings.HasPrefix(NoTC, "Info AppManager") && strings.Contains(NoTC, "Saving to") {
		if !cfg.Local.HideAutosaves {
			savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
			savmatch := savreg.FindStringSubmatch(NoTC)
			if len(savmatch) > 1 {
				if !cfg.Local.HideAutosaves {
					buf := fmt.Sprintf("`%v` 💾 %s", fact.GetGameTime(), savmatch[1])
					fact.LogCMS(cfg.Local.ChannelData.ChatID, buf)
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

			fact.GameMapLock.Lock()
			fact.GameMapName = filename
			fact.GameMapPath = fullpath
			fact.GameMapLock.Unlock()

			cwlog.DoLogCW(fmt.Sprintf("Map saved as: " + filename))
			fact.LastSaveName = filename

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
			cwlog.DoLogGame(line)
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
		cwlog.DoLogGame(line)

		fact.CMS(cfg.Local.ChannelData.ChatID, NoTC)
		/* Lock error */
		if strings.Contains(NoTC, "Couldn't acquire exclusive lock") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio is already running.")
			fact.SetAutoStart(false)
			fact.SetFactorioBooted(false)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Mod Errors */
		if strings.Contains(NoTC, "caused a non-recoverable error.") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio crashed.")
			fact.SetFactorioBooted(false)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Stack traces */
		if strings.Contains(NoTC, "Hosting multiplayer game failed") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio was unable to launch.")
			fact.SetAutoStart(false)
			fact.SetFactorioBooted(false)
			fact.SetFactRunning(false, true)
			return true
		}
		/* level.dat */
		if strings.Contains(NoTC, "level.dat not found.") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
			fact.SetAutoStart(false)
			fact.SetFactorioBooted(false)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Stack traces */
		if strings.Contains(NoTC, "Unexpected error occurred.") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "Factorio crashed.")
			fact.SetFactorioBooted(false)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Multiplayer manger */
		if strings.Contains(NoTC, "MultiplayerManager failed:") {
			if strings.Contains(NoTC, "info.json not found") {
				fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
				fact.SetAutoStart(false)
				fact.SetFactorioBooted(false)
				fact.SetFactRunning(false, true)
				return true
			}
			/*Error ServerMultiplayerManager.cpp:91: MultiplayerManager failed: "Opening zip /home/dist/github/fact-t/saves/_autosave949.tmp.zip failed: Bad zip file"*/
			/* Bad zip file */
			if strings.Contains(NoTC, "failed: Bad zip file") {
				if numwords > 6 {
					if strings.HasPrefix(
						words[7],
						cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactorioHomePrefix+
							cfg.Local.ServerCallsign) &&
						(strings.HasSuffix(words[7], ".zip") || strings.HasSuffix(words[7], ".tmp.zip")) {
						err := os.Remove(words[7])
						if err != nil {
							cwlog.DoLogCW("Unable to remove bad zip file: " + words[7])
							fact.SetAutoStart(false)
						} else {
							cwlog.DoLogCW("Removed bad zip file: " + words[7])
						}
						return true
					}
				}
			}
			/* Corrupt savegame */
			if strings.Contains(NoTC, "Closing file") {
				fact.GameMapLock.Lock()
				path := fact.GameMapPath
				fact.GameMapLock.Unlock()

				var tempargs []string
				tempargs = append(tempargs, "-f")
				tempargs = append(tempargs, path)

				out, errs := exec.Command(cfg.Global.PathData.RMPath, tempargs...).Output()

				if errs != nil {
					cwlog.DoLogCW(fmt.Sprintf("Unabled to delete corrupt savegame. Details:\nout: %v\nerr: %v", string(out), errs))
					fact.SetAutoStart(false)
					fact.CMS(cfg.Local.ChannelData.ChatID, "Unable to load save-game.")
				} else {
					cwlog.DoLogCW("Deleted corrupted savegame.")
					fact.CMS(cfg.Local.ChannelData.ChatID, "Save-game corrupted, performing roll-back.")
				}

				fact.SetFactorioBooted(false)
				fact.SetFactRunning(false, true)
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
		cwlog.DoLogGame(line)

		if NoDSlistlen > 1 {
			NoDSlist[1] = strings.Replace(NoDSlist[1], ":", "", -1)
			pname := NoDSlist[1]

			if pname != "<server>" {

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
					fbuf = fmt.Sprintf("`%v` **%s**: %s", fact.GetGameTime(), factname, cmess)
				} else {
					fbuf = fmt.Sprintf("`%v` %s: %s", fact.GetGameTime(), factname, cmess)
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
					err := disc.SmartWriteDiscordEmbed(cfg.Local.ChannelData.ChatID, myembed)
					if err != nil {
						/* On failure, send normal message */
						cwlog.DoLogCW("Failed to send chat embed.")
					} else {
						/* Stop if succeeds */
						return true
					}
				}
				fact.CMS(cfg.Local.ChannelData.ChatID, fbuf)
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
	if strings.HasPrefix(line, "~") {
		cwlog.DoLogGame(line)
		if strings.Contains(line, "Online:") {
			fact.CMS(cfg.Local.ChannelData.ChatID, "`"+line+"`")
			return true
		}
	}
	return false
}