package support

import (
	"fmt"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func handleActMsg(input *handleData) bool {
	/******************
	 * ACT AREA
	 * Used for logs, and to attempt to warn of potential griefing
	 ******************/

	if strings.HasPrefix(input.line, "[ERROR]") {
		cwlog.DoLogCW(input.line)
		return true

	} else if strings.HasPrefix(input.line, "[ACT]") || strings.HasPrefix(input.line, "[TODO]") {
		cwlog.DoLogGame(input.line)
		if input.wordListLen > 2 {

			pname := input.wordList[1]
			action := input.wordList[2]

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
						} else if strings.Contains(action, "decon") {
							p.SusScore += 2
						}

						thresh := int64(constants.SusWarningThresh)
						if p.Level > 0 {
							thresh += 1000
						}
						if p.SusScore > thresh {

							if time.Since(glob.LastSusWarning) > time.Minute*constants.SusWarningInterval {
								glob.LastSusWarning = time.Now()

								if !cfg.Global.Options.ShutupSusWarn {
									suspect := "Possible suspicious activity: " + pname + "\n"

									serverTag := fmt.Sprintf("%v-%v", cfg.Local.Callsign, cfg.Local.Name)
									if cfg.Local.Channel.ChatChannel != "" {
										serverTag = fmt.Sprintf("<#%v> [%v]\n", cfg.Local.Channel.ChatChannel, serverTag)
									}
									logURL := ""
									if cfg.GetGameLogURL() != "" {
										logURL = "Log: " + cfg.GetGameLogURL() + "\n"
									}
									pingTag := ""
									if cfg.Global.Discord.SusPingRole != "" {
										pingTag = fmt.Sprintf("\n<@&%v>", cfg.Global.Discord.SusPingRole)
									}

									buf := serverTag + suspect + logURL + pingTag
									fact.ReportStatus(buf)

									fact.FactChat(suspect)
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
			fact.SetLastBan(trustname)

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

func handleUnBan(input *handleData) bool {
	/******************
	 * UNBAN
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[UNBANNED]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			trustname := input.noDatestampList[1]
			fact.SetLastBan("")

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
