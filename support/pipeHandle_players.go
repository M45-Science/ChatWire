package support

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

func handleOnlinePlayers(input *handleData) bool {
	/* ********************************************************
	 * CAPTURE ONLINE PLAYERS
	 * Only used for servers that are not using our soft mod
	 **********************************************************/
	if strings.HasPrefix(input.line, "Online players") {

		if input.wordListLen > 2 {
			prevCount := fact.NumPlayers
			poc := strings.Join(input.wordList[2:], " ")
			poc = strings.ReplaceAll(poc, "(", "")
			poc = strings.ReplaceAll(poc, ")", "")
			poc = strings.ReplaceAll(poc, ":", "")
			poc = strings.ReplaceAll(poc, " ", "")

			nump, _ := strconv.Atoi(poc)
			fact.NumPlayers = (nump)

			fact.UpdateChannelName()
			// If the last player logs out, update immediately so the channel name
			// doesn't sit stale behind the normal cooldown.
			if prevCount != 0 && nump == 0 {
				fact.DoUpdateChannelNameForce()
			}
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
		fact.WriteFact(glob.OnlineCommand)
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {

			//Check ban list, promote, show join, unpause, patreon/nitro, online command
			pname := sclean.UnicodeCleanup(input.noDatestampList[1])
			if banlist.CheckBanList(pname, true) {
				return true
			}
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
			if glob.SoftModVersion != constants.Unknown {
				UpdateDuration()

				/* Give people patreon/nitro tags in-game. */
				did := disc.GetDiscordIDFromFactorioName(pname)
				if did != "" {
					if disc.IsPatreon(did) {
						fact.WriteFact("/patreon %s", pname)
					}
					if disc.IsNitro(did) {
						fact.WriteFact("/nitro %s", pname)
					}
				}
			}

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
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
			}

			// Refresh player count on leave events even when the soft-mod is active.
			// Some servers auto-pause when empty, which can prevent the periodic /online
			// poll from running and leave the channel name stuck with a stale count.
			fact.WriteFact(glob.OnlineCommand)

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(pname)
		}
		return true
	}
	return false
}

var lastConnectTime time.Time
var lastConnector string

func handleIncomingAnnounce(input *handleData) bool {
	/********************************
	 * Announce incoming connections
	 ********************************/
	if strings.Contains(input.noTimecode, "Queuing ban recommendation check for user ") {
		if input.trimmedWordsLen > 1 {
			pName := input.trimmedWords[input.trimmedWordsLen-1]

			dmsg := fmt.Sprintf("`%v` %v is connecting.", fact.Gametime, pName)
			fmsg := fmt.Sprintf("%v is connecting.", pName)
			cwlog.DoLogGame(dmsg)

			if time.Since(lastConnectTime) > time.Second*30 || lastConnector != pName {
				fact.FactChat(fmsg)
				fact.CMS(cfg.Local.Channel.ChatChannel, dmsg)
			}

			lastConnectTime = time.Now()
			lastConnector = pName

			glob.PausedLock.Lock()
			if glob.PausedForConnect {
				if strings.EqualFold(glob.PausedFor, pName) {
					glob.PausedConnectAttempt = true
					fact.WriteFact("/aspeed 4")
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

		prevCount := fact.NumPlayers

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
					pafk := ""
					if len(fields) > 4 {
						pafk = fields[4]
					}

					plevel := fact.StringToLevel(ptype)

					if pname != "" {
						/* Mark as seen, async */
						go fact.UpdateSeen(pname)

						/* Check if user is banned */
						banlist.CheckBanList(pname, false)

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
				if fact.NumPlayers != prevCount {
					fact.UpdateChannelName()
				}
				return true
			}
		}

		/* Otherwise clear list */
		fact.NumPlayers = (0)
		fact.OnlinePlayersLock.Lock()
		glob.OnlinePlayers = []glob.OnlinePlayerData{}
		fact.OnlinePlayersLock.Unlock()
		if prevCount != 0 {
			fact.UpdateChannelName()
			// Last player left; force an immediate channel name refresh.
			fact.DoUpdateChannelNameForce()
		}

		return true
	}
	return false
}
