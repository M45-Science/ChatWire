package support

import (
	"bufio"
	"strings"
	"time"

	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

type funcList struct {
	name     string
	function func(input *handleData) bool
}

var handleList = []funcList{
	{name: "handleDisconnect", function: handleDisconnect},
	{name: "handleGameTime", function: handleGameTime},
	{name: "handleOnlinePlayers", function: handleOnlinePlayers},
	{name: "handlePlayerJoin", function: handlePlayerJoin},
	{name: "handlePlayerLeave", function: handlePlayerLeave},
	{name: "handleMapLoad", function: handleMapLoad},
	{name: "handleBan", function: handleBan},
	{name: "handleSVersion", function: handleSVersion},
	{name: "handleUnBan", function: handleUnBan},
	{name: "handleFactGoodbye", function: handleFactGoodbye},
	{name: "handleFactReady", function: handleFactReady},
	{name: "handleIncomingAnnounce", function: handleIncomingAnnounce},
	{name: "handleFactVersion", function: handleFactVersion},
	{name: "handleSaveMsg", function: handleSaveMsg},
	{name: "handleExitSave", function: handleExitSave},
	{name: "handleDesync", function: handleDesync},
	{name: "handleCrashes", function: handleCrashes},
}

var softModHandleList = []funcList{
	{name: "handleCmdMsg", function: handleCmdMsg},
	{name: "handleActMsg", function: handleActMsg},
	{name: "handleOnlineMsg", function: handleOnlineMsg},
	{name: "handleSoftModMsg", function: handleSoftModMsg},
	{name: "handlePlayerReport", function: handlePlayerReport},
	{name: "handlePlayerRegister", function: handlePlayerRegister},
	{name: "handleIdiots", function: handleIdiots},
	{name: "handleChatMsg", function: handleChatMsg},
}

var nonChatHandleList = []funcList{
	{name: "handleIdiots", function: handleIdiots},
	{name: "handleChatMsg", function: handleChatMsg},
}

type handleData struct {
	line, lowerCaseLine, NoTC, NoDS                                   string
	lineList, lowerCaseList, NoTClist, NoDSlist, words                []string
	numwords, NoDSlistlen, lowerCaseListlen, NoTClistlen, lineListlen int
}

/*  Chat pipes in-game chat to Discord, and handles log events */
func HandleChat() {

	/* Don't log if the game isn't set to run */
	for glob.ServerRunning {
		time.Sleep(time.Millisecond * 100)

		/* Check if there is anything in the input buffer */
		if fact.GameBuffer != nil {
			reader := bufio.NewScanner(fact.GameBuffer)

			time.Sleep(time.Millisecond * 100)

			for reader.Scan() {
				if !fact.FactIsRunning {
					break
				}
				readLine := reader.Text()

				line := sclean.UnicodeCleanup(readLine)

				/* Reject short lines */
				ll := len(line)
				if ll <= 0 {
					continue
				}
				/* We have input, server is alive */
				fact.SetFactRunning(true)

				/*
				 * Timecode removal, split into words, save lengths
				 */

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

				//We pass this via pointer to all the handler functions
				input := &handleData{
					line, lowerCaseLine, NoTC, NoDS,
					lineList, lowerCaseList, NoTClist, NoDSlist, words,
					numwords, NoDSlistlen, lowerCaseListlen, NoTClistlen, lineListlen,
				}

				/* Decrement every time we see activity, if we see time not progressing, add two */
				if fact.PausedTicks > 0 {
					fact.PausedTicks--
				}

				/*********************************
				 * FILTERED AREA
				 * NO CONSOLE CHAT
				 **********************************/
				if !strings.HasPrefix(line, "<server>") {

					/*********************************
					 * NO CHAT OR COMMAND LOG AREA
					 *********************************/
					if !strings.HasPrefix(NoDS, "[CHAT]") && !strings.HasPrefix(NoDS, "[SHOUT]") && !strings.HasPrefix(line, "[CMD]") {

						/*
						 * Standard handles
						 */
						for _, handle := range handleList {
							if handle.function(input) {
								continue
							}
						}

						/*
						 * Softmod only
						 */
						if glob.SoftModVersion != constants.Unknown {
							for _, handle := range softModHandleList {
								if handle.function(input) {
									continue
								}
							}
						}

					} else {

						/*
						 * Non-chat only
						 */
						for _, handle := range nonChatHandleList {
							if handle.function(input) {
								continue
							}
						}
					}
				}
			}
		}
	}
}
