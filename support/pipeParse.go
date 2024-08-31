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

var noChatHandles = []funcList{
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
}

var chatHandles = []funcList{
	{function: handleIdiots},
	{function: handleChatMsg},
}

type handleData struct {
	line, lowerLine, timecode, datestamp                                       string
	lineList, lowerLineList, noTimecodeList, noDatestampList, words            []string
	numWords, noDatestampListLen, lowerListLen, noTimecodeListLen, lineListLen int
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
				rawLine := sclean.UnicodeCleanup(readLine)

				/* Reject short lines */
				ll := len(rawLine)
				if ll == 0 {
					continue
				}
				/* We have input, server is alive */
				fact.SetFactRunning(true)

				input := preProcessFactorioOutput(rawLine)

				/* Decrement every time we see activity, if we see time not progressing, add two */
				if fact.PausedTicks > 0 {
					fact.PausedTicks--
				}

				/*********************************
				 * FILTERED AREA
				 * NO CONSOLE CHAT
				 **********************************/
				if !strings.HasPrefix(input.line, "<server>") {

					/*********************************
					 * NO CHAT OR COMMAND LOG AREA
					 *********************************/
					if !strings.HasPrefix(input.datestamp, "[CHAT]") && !strings.HasPrefix(input.datestamp, "[SHOUT]") && !strings.HasPrefix(input.line, "[CMD]") {

						/*
						 * No-chat handles
						 */
						for _, handle := range noChatHandles {
							if handle.function(input) {
								continue
							}
						}

						/*
						 * Soft-mod only
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
						 * Chat only
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

func preProcessFactorioOutput(line string) *handleData {
	/*
	 * Timecode removal, split into words, save lengths
	 */

	trimmed := strings.TrimLeft(line, " ")
	words := strings.Split(trimmed, " ")
	numWords := len(words)
	timecode := constants.Unknown
	datestamp := constants.Unknown

	if numWords > 1 {
		timecode = strings.Join(words[1:], " ")
	}
	if numWords > 2 {
		datestamp = strings.Join(words[2:], " ")
	}

	/* Separate args -- for use with script output */
	lineList := strings.Split(line, " ")
	lineListLen := len(lineList)

	/* Separate args, no timecode -- for use with Factorio subsystem output */
	noTimecodeList := strings.Split(timecode, " ")
	noTimecodeListLen := len(noTimecodeList)

	/* Separate args, no datestamp -- for use with normal Factorio log output */
	noDatestampList := strings.Split(datestamp, " ")
	noDatestampListLen := len(noDatestampList)

	/* Lowercase converted */
	lowerCaseLine := strings.ToLower(line)
	lowerCaseList := strings.Split(lowerCaseLine, " ")
	lowerCaseListlen := len(lowerCaseList)

	return &handleData{
		line, lowerCaseLine, timecode, datestamp,
		lineList, lowerCaseList, noTimecodeList, noDatestampList, words,
		numWords, noDatestampListLen, lowerCaseListlen, noTimecodeListLen, lineListLen,
	}
}
