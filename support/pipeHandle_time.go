package support

import (
	"fmt"
	"strconv"
	"strings"

	"ChatWire/constants"
	"ChatWire/fact"
)

func handleGameTime(input *handleData) bool {
	/******************************************************
	 * GET FACTORIO TIME
	 * While this is needed for games without our softmod,
	 * we should be using tick count instead.
	 ******************************************************/
	if strings.Contains(input.lowerLine, " second") || strings.Contains(input.lowerLine, " minute") || strings.Contains(input.lowerLine, " hour") || strings.Contains(input.lowerLine, " day") {

		day := 0
		hour := 0
		minute := 0
		second := 0

		if input.lowerListLen > 1 {

			for x := 0; x < input.lowerListLen; x++ {
				if strings.Contains(input.lowerWordList[x], "day") {
					day, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "hour") {
					hour, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "minute") {
					minute, _ = strconv.Atoi(input.lowerWordList[x-1])
				} else if strings.Contains(input.lowerWordList[x], "second") {
					second, _ = strconv.Atoi(input.lowerWordList[x-1])
				}
			}

			var newtime string
			if day > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)
			} else if hour > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d-%.2d", hour, minute, second)
			} else if minute > 0 {
				newtime = fmt.Sprintf("%.2d-%.2d", minute, second)
			} else {
				newtime = fmt.Sprintf("%.2d", second)
			}

			/* Don't add the time if we are slowed down for players connecting, or paused */
			if fact.SlowConnectTimer == 0 && fact.PausedTicks <= 2 {
				fact.TickHistoryLock.Lock()
				fact.TickHistory = append(fact.TickHistory,
					fact.TickInt{Day: day, Hour: hour, Min: minute, Sec: second})

				/* Chop old tick history */
				thl := len(fact.TickHistory) - fact.MaxTickHistory
				if thl > 0 {
					fact.TickHistory = fact.TickHistory[thl:]
				}
				fact.TickHistoryLock.Unlock()
			}

			if fact.LastGametime == fact.Gametime {
				if fact.PausedTicks <= constants.PauseThresh {
					fact.PausedTicks = fact.PausedTicks + 2
				}
			} else {
				fact.PausedTicks = 0
			}
			fact.LastGametime = fact.Gametime
			fact.GametimeString = input.lowerLine
			fact.Gametime = newtime
		}
	}
	/* This might block input by accident, don't do it */
	return false
}
