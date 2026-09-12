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
			prevCount := fact.NumPlayersCurrent()
			poc := strings.Join(input.wordList[2:], " ")
			poc = strings.ReplaceAll(poc, "(", "")
			poc = strings.ReplaceAll(poc, ")", "")
			poc = strings.ReplaceAll(poc, ":", "")
			poc = strings.ReplaceAll(poc, " ", "")

			nump, _ := strconv.Atoi(poc)
			fact.SetNumPlayers(nump)

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
		fact.RequestOnlinePlayers()
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {

			//Check ban list, promote, show join, unpause, patreon/nitro, online command
			pname := sclean.UnicodeCleanup(input.noDatestampList[1])
			if banlist.CheckBanList(pname, true) {
				return true
			}
			plevelname := fact.AutoPromote(pname, false, true)

			pname = sclean.EscapeDiscordMarkdown(pname)

			buf := fmt.Sprintf("`%v` **%s joined**%s", fact.CurrentGametime(), pname, plevelname)

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
						fact.WriteSoftModSpeed(cfg.Local.Options.Speed)
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
						fact.WriteSoftModCommand("supporter", map[string]any{"name": pname, "patreon": true})
					}
					if disc.IsNitro(did) {
						fact.WriteSoftModCommand("supporter", map[string]any{"name": pname, "nitro": true})
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
			fact.RequestOnlinePlayers()

			fact.UpdateSeen(pname)
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

			dmsg := fmt.Sprintf("`%v` %v is connecting.", fact.CurrentGametime(), pName)
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
					fact.WriteSoftModSpeed(4.0 / 60.0)
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
