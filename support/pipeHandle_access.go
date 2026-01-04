package support

import (
	"fmt"
	"strings"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

/* Protect players from dumb mistakes with registration codes */
func handleIdiots(input *handleData) bool {
	if ProtectIdiots(input.line) {
		buf := "You did not enter that as a command!\nYou have posted your registration code publicly.\nTo protect you, the code has been invalidated.\nPlease try again, and read the directions more carefully!"
		fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, buf)
		return true
	}

	return false
}

func handlePlayerReport(input *handleData) bool {
	/******************
	 * Player REPORT
	 ******************/
	if strings.HasPrefix(input.line, "[REPORT]") {
		cwlog.DoLogGame(input.line)
		if input.wordListLen >= 3 {
			reporter := "Reporter: " + input.wordList[1] + "\n"
			msg := strings.Join(input.wordList[2:], " ") + "\n"

			serverTag := fmt.Sprintf("%v-%v", cfg.Local.Callsign, cfg.Local.Name)
			if cfg.Local.Channel.ChatChannel != "" {
				serverTag = fmt.Sprintf("<#%v> [%v]\n", cfg.Local.Channel.ChatChannel, serverTag)
			}
			logURL := ""
			if cfg.GetGameLogURL() != "" {
				logURL = "Log: " + cfg.GetGameLogURL() + "\n"
			}
			pingTag := ""
			if cfg.Global.Discord.BanishPingRole != "" {
				pingTag = fmt.Sprintf("\n<@&%v>", cfg.Global.Discord.BanishPingRole)
			}

			buf := serverTag + reporter + msg + logURL + pingTag
			fact.LogCMS(cfg.Global.Discord.ReportChannel, buf)

			fact.FactChat("REPORT: %v: %v", reporter, msg)
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
						fact.FactWhisper(pname, "[SYSTEM] This is not the correct server for entering registration codes! You need to connect to %v-%v to use that command. Please read the directions more carefully...",
							cfg.Global.GroupName, cfg.Global.PrimaryServer)
						return true
					}

					if strings.EqualFold(discid, pid) && strings.EqualFold(factname, pname) {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', wants to register a few times... just to be sure.", pname))
						fact.FactWhisper(pname, "[SYSTEM] This Factorio user, and discord user are already connected! You do not need to re-register...")
						codegood = true
						/* Do not break, process */
					} else if discid != "" && discid != "0" {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', tried to register a discord user that is already registered to different Factorio player.", pname))
						fact.FactWhisper(pname, "[SYSTEM] That discord user is already connected to a different Factorio user... Unable to complete registration.")
						codegood = false
						continue
					} else if factname != "" {
						fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Factorio player '%s', tried to register a Factorio user that is already registered to a different discord user.", pname))
						fact.FactWhisper(pname, "[SYSTEM] This Factorio user is already connected to a different discord user... Unable to complete registration.")
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
								fact.FactWhisper(pname, "[SYSTEM] Sorry, there was an internal error. I could not find the discord role '%s'. The moderatators will be informed of the issue.", newrole)
								continue
							}

							erradd := disc.SmartRoleAdd(cfg.Global.Discord.Guild, pid, regrole.ID)

							if erradd != nil || disc.DS == nil {
								fact.LogCMS(cfg.Global.Discord.ReportChannel, fmt.Sprintf("Register: Could not assign role '%v'. Requested for user '%v'.", newrole, pname))
								fact.FactWhisper(pname, "[SYSTEM] Sorry, there was an internal error. I could not assign discord role '%s'. The moderatators will be informed of the issue.", newrole)
								continue
							}
							fact.FactWhisper(pname, "[SYSTEM] Registration complete!")
							fact.LogGameCMS(true, cfg.Global.Discord.ReportChannel, fmt.Sprintf("Registered player: %v", pname))
							continue
						} else {
							fact.FactWhisper(pname, "[SYSTEM] Sorry, I couldn't find the discord guild info! The moderators will be informed of the issue.")
							fact.LogCMS(cfg.Global.Discord.ReportChannel, "Register: Unable to get discord guild info!")
							continue
						}
					}
					continue
				}
			} /* End of loop */
			glob.PasswordListLock.Unlock()
			if !codefound {
				cwlog.DoLogCW("Register: Factorio player '%s' tried to use an invalid or expired code.", pname)
				fact.FactWhisper(pname, "[SYSTEM] Sorry, that code is invalid or expired.")
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
