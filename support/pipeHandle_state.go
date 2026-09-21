package support

import (
	"regexp"
	"strings"
	"time"

	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func isFactorioReadyLine(line string) bool {
	line = strings.TrimSpace(line)
	return line != "" && strings.Contains(line, "Starting RCON interface")
}

func handleFactGoodbye(input *handleData) bool {
	/******************
	 * GOODBYE
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Goodbye") {
		fact.SetLastBan("")
		fact.NotifyFactorioGoodbye(input.generation)
		return true
	}
	return false
}

func handleFactReady(input *handleData) bool {
	/*****************
	 * READY MESSAGE
	 ******************/
	if isFactorioReadyLine(input.noTimecode) {
		fact.NotifyFactorioProgress(input.generation, "rcon-ready", "")
		fact.WriteAdminlist()

		// A Factorio boot implies no players online yet; clear any stale count so the
		// Discord channel name can be refreshed immediately.
		fact.SetNumPlayers(0)
		fact.OnlinePlayersLock.Lock()
		glob.OnlinePlayers = []glob.OnlinePlayerData{}
		fact.OnlinePlayersLock.Unlock()

		fact.NotifyFactorioReady(input.generation)

		newHist := modupdate.ModHistoryItem{Name: modupdate.BootName, Date: time.Now(), InfoItem: true}
		modupdate.AddModHistory(newHist)

		// Initialize time immediately on saves without SoftMod while also
		// starting the normal machine-protocol handshake when it is available.
		requestFactorioStatus()
	}
	return false
}

func handleFactVersion(input *handleData) bool {
	/* **********************
	 * GET FACTORIO VERSION
	 ***********************/
	if strings.HasPrefix(input.noTimecode, "Loading mod base") {
		fact.NotifyFactorioProgress(input.generation, "mod-load", modLoadStatusDetail(input.noTimecode))
		//cwlog.DoLogCW(input.noTimecode)
		if input.noTimecodeListLen > 3 {
			fact.SetFactorioVersion(input.noTimecodeList[3])
		}
	} else if strings.HasPrefix(input.noTimecode, "Loading mod ") {
		fact.NotifyFactorioProgress(input.generation, "mod-load", modLoadStatusDetail(input.noTimecode))
	}
	return false
}

var modLoadCountPattern = regexp.MustCompile(`\((\d+/\d+)\)`)

func modLoadStatusDetail(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "Loading mod ") {
		return ""
	}

	modInfo := strings.TrimSpace(strings.TrimPrefix(line, "Loading mod "))
	if modInfo == "" {
		return ""
	}

	count := ""
	if match := modLoadCountPattern.FindStringSubmatch(modInfo); len(match) > 1 {
		count = match[1]
		modInfo = strings.Replace(modInfo, match[0], "", 1)
	}

	fields := strings.Fields(modInfo)
	if len(fields) == 0 {
		return ""
	}

	name := fields[0]
	if count != "" {
		return count + " " + name
	}
	return name
}
