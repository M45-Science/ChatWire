package support

import (
	"strings"

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
	{function: handleChatWire},
	{function: handleGameTime},
	{function: handleOnlinePlayers},
	{function: handlePlayerJoin},
	{function: handlePlayerLeave},
	{function: handleMapLoad},
	{function: handleBan},
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

type handleData struct {
	generation                                                                        uint64
	line, lowerLine, noTimecode, noDatestamp                                          string
	wordList, lowerWordList, noTimecodeList, noDatestampList, trimmedWords            []string
	trimmedWordsLen, noDatestampListLen, lowerListLen, noTimecodeListLen, wordListLen int
}

func runHandles(handles []funcList, input *handleData) {
	for _, handle := range handles {
		if handle.function(input) {
			return
		}
	}
}

/*  Chat pipes in-game chat to Discord, and handles log events */
func HandleChat() {
	ctx := glob.RuntimeContext()

	/* Don't log if the game isn't set to run */
	for {
		lines, generation := fact.GameOutputCurrent()
		if lines == nil {
			select {
			case <-ctx.Done():
				return
			case <-fact.GameOutputChanged():
			}
			continue
		}

		var readLine string
		var ok bool
		select {
		case readLine, ok = <-lines:
			if !ok {
				currentLines, currentGeneration := fact.GameOutputCurrent()
				if currentLines == lines && currentGeneration == generation {
					select {
					case <-ctx.Done():
						return
					case <-fact.GameOutputChanged():
					}
				}
				continue
			}
		case <-fact.GameOutputChanged():
			continue
		case <-ctx.Done():
			return
		}
		if generation != 0 && !fact.IsCurrentFactorioGeneration(generation) {
			continue
		}
		rawLine := sclean.UnicodeCleanup(readLine)

		/* We have input, server is alive */
		//fact.SetFactRunning(true, false)
		glob.ResetNoResponseCount()

		/* Decrement every time we see activity, if we see time not progressing, add two */
		if fact.PausedTicks > 0 {
			fact.PausedTicks--
		}

		/* Reject short lines */
		if rawLine == "" {
			continue
		}

		input := preProcessFactorioOutput(rawLine)
		input.generation = generation

		/*********************************
		 * FILTERED AREA
		 * NO CONSOLE CHAT
		 **********************************/
		if !strings.HasPrefix(input.line, "<server>") {

			/*********************************
			 * NO CHAT OR COMMAND LOG AREA
			 *********************************/
			if !strings.HasPrefix(input.noDatestamp, "[CHAT]") && !strings.HasPrefix(input.noDatestamp, "[SHOUT]") {

				/*
				 * No-chat handles
				 */
				runHandles(noChatHandles, input)

			} else {

				/*
				 * Chat only
				 */
				/* Dont send leaked register codes to chat, warn */
				if !handleIdiots(input) {
					handleChatMsg(input)
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
	trimmedWords := strings.Split(trimmed, " ")
	trimmedWordsLen := len(trimmedWords)
	noTimecode := constants.Unknown
	noDatestamp := constants.Unknown

	if trimmedWordsLen > 1 {
		noTimecode = strings.Join(trimmedWords[1:], " ")
	}
	if trimmedWordsLen > 2 {
		noDatestamp = strings.Join(trimmedWords[2:], " ")
	}

	/* Separate args -- for use with script output */
	wordList := strings.Split(line, " ")
	wordListLen := len(wordList)

	/* Separate args, no timecode -- for use with Factorio subsystem output */
	noTimecodeList := strings.Split(noTimecode, " ")
	noTimecodeListLen := len(noTimecodeList)

	/* Separate args, no datestamp -- for use with normal Factorio log output */
	noDatestampList := strings.Split(noDatestamp, " ")
	noDatestampListLen := len(noDatestampList)

	/* Lowercase converted */
	lowerLine := strings.ToLower(line)
	lowerWordList := strings.Split(lowerLine, " ")
	lowerWordListLen := len(lowerWordList)

	return &handleData{
		0, line, lowerLine, noTimecode, noDatestamp,
		wordList, lowerWordList, noTimecodeList, noDatestampList, trimmedWords,
		trimmedWordsLen, noDatestampListLen, lowerWordListLen, noTimecodeListLen, wordListLen,
	}
}
