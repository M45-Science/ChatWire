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
	function func(input *handleData) bool
}

var nonChatHandles = []funcList{
	{function: handleDisconnect},
	{function: handleGameTime},
	{function: handleOnlinePlayers},
	{function: handlePlayerJoin},
	{function: handlePlayerLeave},
	{function: handleMapLoad},
	{function: handleBan},
	{function: handleSVersion},
	{function: handleUnBan},
	{function: handleFactGoodbye},
	{function: handleFactReady},
	{function: handleIncomingAnnounce},
	{function: handleFactVersion},
	{function: handleSaveMsg},
	{function: handleExitSave},
	{function: handleDesync},
	{function: handleCrashes},
}

var softModHandles = []funcList{
	{function: handleCmdMsg},
	{function: handleActMsg},
	{function: handleOnlineMsg},
	{function: handleSoftModMsg},
	{function: handlePlayerReport},
	{function: handlePlayerRegister},
	{function: handleIdiots},
	{function: handleChatMsg},
}

var chatHandles = []funcList{
	{function: handleIdiots},
	{function: handleChatMsg},
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
						for _, handle := range nonChatHandles {
							if handle.function(input) {
								continue
							}
						}

						/*
						 * Softmod only
						 */
						if glob.SoftModVersion != constants.Unknown {
							for _, handle := range softModHandles {
								if handle.function(input) {
									continue
								}
							}
						}

					} else {

						/*
						 * Non-chat only
						 */
						for _, handle := range chatHandles {
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
