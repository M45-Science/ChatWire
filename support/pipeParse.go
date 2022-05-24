package support

import (
	"bufio"
	"strings"
	"time"

	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

/*  Chat pipes in-game chat to Discord, and handles log events */
func HandleChat() {

	for glob.ServerRunning {
		if fact.GameBuffer != nil {
			reader := bufio.NewScanner(fact.GameBuffer)
			time.Sleep(time.Millisecond * 100)
			for reader.Scan() {
				if !fact.IsFactRunning() {
					break
				}
				line := reader.Text()
				/* Remove return/newline */
				line = strings.TrimSuffix(line, "\r")
				line = strings.TrimSuffix(line, "\n")

				/* Reject short lines */
				ll := len(line)
				if ll <= 0 {
					continue
				}
				/* Server is alive */
				fact.SetFactRunning(true)

				/* Timecode removal */
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

				/* Separate args -- for use with script output */
				lineList := strings.Split(line, " ")
				lineListlen := len(lineList)

				/* Separate args, notc -- for use with Factorio subsystem output */
				NoTClist := strings.Split(NoTC, " ")
				NoTClistlen := len(NoTClist)

				/* Separate args, nods -- for use with normal Factorio log output */
				NoDSlist := strings.Split(NoDS, " ")
				NoDSlistlen := len(NoDSlist)

				/* Lowercase converted */
				lowerCaseLine := strings.ToLower(line)
				lowerCaseList := strings.Split(lowerCaseLine, " ")
				lowerCaseListlen := len(lowerCaseList)

				/* Decrement every time we see activity, if we see time not progressing, add two */
				fact.PausedTicksLock.Lock()
				if fact.PausedTicks > 0 {
					fact.PausedTicks--
				}
				fact.PausedTicksLock.Unlock()

				/*********************************
				 * FILTERED AREA
				 * NO ESCAPED OR CONSOLE CHAT
				 **********************************/
				if !strings.HasPrefix(line, "<server>") {

					/******************
					 * NO CHAT AREA
					 ******************/
					if !strings.HasPrefix(NoDS, "[CHAT]") && !strings.HasPrefix(NoDS, "[SHOUT]") && !strings.HasPrefix(line, "[CMD]") {

						/* Don't eat event, this is capable of eating random text */
						go handleGameTime(lowerCaseLine, lowerCaseList, lowerCaseListlen)

						if handlePlayerReport(line, lineList, lowerCaseListlen) {
							continue
						}

						if handlePlayerRegister(line, lineList, lineListlen) {
							continue
						}

						if handleOnlinePlayers(line, lineList, lineListlen) {
							continue
						}

						if handlePlayerJoin(NoDS, NoDSlist, NoDSlistlen) {
							continue
						}

						if handlePlayerLeave(NoDS, line, NoDSlist, lowerCaseListlen) {
							continue
						}

						if handleSoftModMsg(line, lineList, lineListlen) {
							continue
						}

						/* Don't eat event, used for fixLockers */
						go handleSlowConnect(NoTC, line)

						if handleMapLoad(NoTC, NoDSlist, NoTClist, NoTClistlen) {
							continue
						}

						if handleModLoad(NoTC) {
							continue
						}

						go handleBan(NoDS, NoDSlist, NoDSlistlen)

						if handleSVersion(line, lineList, lineListlen) {
							continue
						}

						if handleUnBan(NoDS, NoDSlist, NoDSlistlen) {
							continue
						}

						if handleFactGoodbye(NoTC) {
							continue
						}

						if handleFactReady(NoTC) {
							continue
						}

						if handleFixLockers(NoTC) {
							continue
						}

						if handleIncomingAnnounce(NoTC, words, numwords) {
							continue
						}

						go handleFactVersion(NoTC, line, NoTClist, NoTClistlen)

						if handleSaveMsg(NoTC) {
							continue
						}

						if handleExitSave(NoTC, NoTClist, NoTClistlen) {
							continue
						}

						if handleDesync(NoTC, line) {
							continue
						}

						if handleCrashes(NoTC, line, words, numwords) {
							continue
						}

						if handleCmdMsg(line) {
							continue
						}
						if handleActMsg(line, lineList, lineListlen) {
							continue
						}
						if handleOnlineMsg(line) {
							continue
						}
					} else {
						/* Protect players from dumb mistakes with registration codes */
						if handleIdiots(line) {
							continue
						}

						if handleChatMsg(NoDS, line, NoDSlist, NoDSlistlen) {
							continue
						}
					}
					/******************
					 * END FILTERED
					 ******************/
				}
			}
		}
	}
}
