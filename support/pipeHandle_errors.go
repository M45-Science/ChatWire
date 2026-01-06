package support

import (
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func handleDesync(input *handleData) bool {
	/******************
	 * CAPTURE DESYNC
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Info") {
		if strings.Contains(input.noTimecode, "DesyncedWaitingForMap") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Desync: "+input.noTimecode)
			cwlog.DoLogCW("desync: " + input.noTimecode)
			newItem := modupdate.ModHistoryItem{InfoItem: true,
				Name: "Desync", Notes: input.noTimecode, Date: time.Now()}
			modupdate.AddModHistory(newItem)

			if !fact.FactorioBootedAt.IsZero() && time.Since(fact.FactorioBootedAt) >= 15*time.Minute && !fact.QueueFactReboot {
				fact.QueueFactReboot = true
					cwlog.DoLogCW("Desync detected after 15m Factorio uptime; reboot queued.")
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Desync detected; Factorio reboot queued.**")
			}
			return true
		}
	}
	return false
}

func handleCrashes(input *handleData) bool {

	/* *****************
	 * CAPTURE CRASHES
	 ******************/
	if strings.HasPrefix(input.line, "__level__/") {
		cwlog.DoLogCW(input.line)
	}
	if strings.HasPrefix(input.noTimecode, "Error") {
		cwlog.DoLogCW(input.noTimecode)

		/* Mod load error */
		if strings.Contains(input.noTimecode, "Failed to load mod") {
			var newItem modupdate.ModHistoryItem

			if input.wordListLen >= 10 {
				modName := input.wordList[10]
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio failed loading game mod: "+modName)
				newItem = modupdate.ModHistoryItem{InfoItem: true,
					Name: "Mod load failed", Notes: input.noTimecode, Date: time.Now()}
			} else {
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio failed loading game mods.")
				newItem = modupdate.ModHistoryItem{InfoItem: true,
					Name: "Loading game mods failed", Notes: input.noTimecode, Date: time.Now()}
			}
			modupdate.AddModHistory(newItem)

			fact.SetAutolaunch(false, true)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Lock error */
		if strings.Contains(input.noTimecode, "Couldn't acquire exclusive lock") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio is already running.")
			fact.SetAutolaunch(false, true)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Mod Errors */
		if strings.Contains(input.noTimecode, "caused a non-recoverable error.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua-script error and will reboot.**")

			newHist := modupdate.ModHistoryItem{Name: "Factorio closed with a lua error.", Notes: input.noTimecode, Date: time.Now(), InfoItem: true}
			modupdate.AddModHistory(newHist)

			fact.SetFactRunning(false, true)
			return true
		}
		/* Stack traces */
		if strings.Contains(input.noTimecode, "Hosting multiplayer game failed") {
			if strings.Contains(input.noTimecode, "directory iterator cannot open directory") {
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio didn't find any save-games.")
			} else {
				fact.LogCMS(cfg.Local.Channel.ChatChannel, "Factorio was unable to load a multiplayer game.")
			}
			fact.SetAutolaunch(false, true)
			fact.SetFactRunning(false, true)
			return true
		}
		/* level.dat */
		if strings.Contains(input.noTimecode, "level.dat not found.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
			fact.SetAutolaunch(false, true)
			fact.SetFactRunning(false, true)
			return true
		}
		/* Stack traces */
		if strings.Contains(input.noTimecode, "Unexpected error occurred.") {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, "**Factorio crashed.**")
			fact.SetFactRunning(false, true)

			newItem := modupdate.ModHistoryItem{InfoItem: true,
				Name: "Factorio crashed", Notes: input.noTimecode, Date: time.Now()}
			modupdate.AddModHistory(newItem)
			return true
		}
		if strings.Contains(input.noTimecode, "CommandLineMultiplayer") {
			if strings.Contains(input.noTimecode, "No latest save file found in") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "No save-game found.")
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
		}
		if strings.Contains(input.noTimecode, "Scenario") && strings.HasSuffix(input.noTimecode, "not found") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Invalid scenario specified, clearing scenario setting.")
			cfg.Local.Settings.Scenario = ""
			fact.SetAutolaunch(false, true)
			fact.SetFactRunning(false, true)
			return true
		}

		/* Multiplayer manger */
		if strings.HasPrefix(input.noTimecode, "Error ServerMultiplayerManager") {
			if time.Since(glob.LastCrashReport) > constants.CrashReportInterval*time.Second {
				glob.LastCrashReport = time.Now()

				/* Suppress connection error messages */
				if !strings.Contains(input.noTimecode, "Matching server connection failed") {
					fact.CMS(cfg.Global.Discord.ReportChannel, cfg.Global.GroupName+"-"+cfg.Local.Callsign+": "+cfg.Local.Name+":\n"+input.noTimecode)
				}
			}
		}
		if strings.Contains(input.noTimecode, "MultiplayerManager failed:") {

			if strings.Contains(input.noTimecode, "Host address is already in use") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "The specified port is already in use: "+strconv.FormatInt(int64(cfg.Local.Port), 10))
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
			if strings.Contains(input.noTimecode, "cannot be loaded because it is higher than the game version") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio version is too old for the save game.**")
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
			if strings.Contains(input.noTimecode, "syntax error") || strings.Contains(input.noTimecode, "unexpected symbol") ||
				strings.Contains(input.noTimecode, "expected") {
				newItem := modupdate.ModHistoryItem{InfoItem: true,
					Name: "Lua Syntax Error", Notes: input.noTimecode, Date: time.Now()}
				modupdate.AddModHistory(newItem)

				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua syntax error and will stop.**")
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
			if strings.Contains(input.noTimecode, "Error while running command") {
				newItem := modupdate.ModHistoryItem{InfoItem: true,
					Name: "Lua Command Error", Notes: input.noTimecode, Date: time.Now()}
				modupdate.AddModHistory(newItem)
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio encountered a lua command error.**")
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
			if strings.Contains(input.noTimecode, "info.json not found") {
				fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to load save-game.")
				fact.SetAutolaunch(false, true)
				fact.SetFactRunning(false, true)
				return true
			}
			/* Bad zip file */
			if strings.Contains(input.noTimecode, "(Bad zip file)") {
				if input.trimmedWordsLen > 6 {
					if strings.HasSuffix(input.trimmedWords[7], ".zip") || strings.HasSuffix(input.trimmedWords[7], ".tmp.zip") {
						err := os.Remove(input.trimmedWords[7])
						if err != nil {
							cwlog.DoLogCW("Unable to remove bad zip file: " + input.trimmedWords[7])
							fact.SetAutolaunch(false, true)
							fact.SetFactRunning(false, true)
							return true
						} else {
							cwlog.DoLogCW("Removed bad zip file: " + input.trimmedWords[7])
						}
						return true
					}
				}
			}
			/* Corrupt savegame */
			if strings.Contains(input.noTimecode, "Closing file") {
				errs := os.Remove(fact.GameMapPath)

				if errs != nil {
					cwlog.DoLogCW("Unable to delete corrupt savegame. Details:\nfile: %v\nerr: %v", fact.GameMapPath, errs)
					fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to remove corrupted save-game.")
					fact.SetAutolaunch(false, true)
					fact.SetFactRunning(false, true)
					return true
				} else {
					cwlog.DoLogCW("Deleted corrupted savegame.")
					fact.CMS(cfg.Local.Channel.ChatChannel, "Save-game corrupted, performing automatic roll-back.")
				}

				fact.SetFactRunning(false, true)
				return true
			}

			if strings.Contains(input.noTimecode, "Exception at tick") {
				newItem := modupdate.ModHistoryItem{InfoItem: true,
					Name: "Factorio Crashed", Notes: input.noTimecode, Date: time.Now()}
				modupdate.AddModHistory(newItem)
				fact.CMS(cfg.Local.Channel.ChatChannel, "**Factorio crashed.**")
				fact.SetFactRunning(false, true)
			}
		}
		return true
	}
	return false
}
