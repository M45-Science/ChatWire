package support

import (
	"strings"
	"time"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func handleSVersion(input *handleData) bool {
	/******************
	 * SVERSION
	 ******************/
	if strings.HasPrefix(input.line, "[SVERSION]") {
		cwlog.DoLogCW(input.line)

		if input.wordListLen > 0 {
			glob.SoftModVersion = input.wordList[1]
			glob.OnlineCommand = constants.SoftModOnlineCMD
			cwlog.DoLogCW("Softmod detected: " + glob.SoftModVersion)
			ConfigSoftMod()
		}
		return true
	}
	return false
}

func handleFactGoodbye(input *handleData) bool {
	/******************
	 * GOODBYE
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Goodbye") {
		fact.SetLastBan("")

		fact.FactorioBootedAt = time.Time{}
		fact.SetFactRunning(false, true)
		return true
	}
	return false
}

func handleFactReady(input *handleData) bool {
	/*****************
	 * READY MESSAGE
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Info RemoteCommandProcessor") && strings.Contains(input.noTimecode, "Starting RCON interface") {
		fact.WriteAdminlist()
		fact.FactorioBooted = true
		fact.FactorioBootedAt = time.Now()
		fact.FactIsRunning = false
		glob.CrashLoopCount = 0
		fact.SetFactRunning(true, true)

		newHist := modupdate.ModHistoryItem{Name: modupdate.BootName, Date: time.Now(), InfoItem: true}
		modupdate.AddModHistory(newHist)

		fact.WriteFact("/sversion")
		fact.WriteFact(glob.OnlineCommand)
	}
	return false
}

func handleFactVersion(input *handleData) bool {
	/* **********************
	 * GET FACTORIO VERSION
	 ***********************/
	if strings.HasPrefix(input.noTimecode, "Loading mod base") {
		//cwlog.DoLogCW(input.noTimecode)
		if input.noTimecodeListLen > 3 {
			fact.FactorioVersion = input.noTimecodeList[3]
		}
	}
	return false
}
