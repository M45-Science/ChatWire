package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

/**************************
 * Show useful info about a server and it's settings
 *************************/
func Info(s *discordgo.Session, i *discordgo.InteractionCreate) {

	verbose := false

	a := i.ApplicationCommandData()
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionBoolean &&
			strings.EqualFold(arg.Name, "verbose") && arg.BoolValue() {
			verbose = true
			break
		}
	}

	buf := "```"
	/* STATS */
	if verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", "ChatWire version", constants.Version)
		if glob.SoftModVersion != constants.Unknown {
			buf = buf + fmt.Sprintf("%17v: %v\n", "SoftMod version", glob.SoftModVersion)
		}
	}
	if fact.FactorioVersion != constants.Unknown {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio version", fact.FactorioVersion)
	}
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	if verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", "ChatWire up-time", tnow.Sub(glob.Uptime.Round(time.Second)).String())

		if !fact.FactorioBootedAt.IsZero() && fact.FactorioBooted {
			buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio up-time", tnow.Sub(fact.FactorioBootedAt.Round(time.Second)).String())
		} else {
			buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio up-time", "not running")
		}
	}

	if verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Save name", fact.LastSaveName)
	}
	if fact.GametimeString != constants.Unknown {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Map time", fact.GametimeString)
	}
	if fact.NumPlayers > 0 || verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Players online", fact.NumPlayers)
	}

	if cfg.Local.Options.Schedule != "" {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Next map reset", fact.NextReset)
		buf = buf + fmt.Sprintf("%17v: %v\n", "Time till reset", fact.TillReset)
	}

	/* SETTINGS */
	buf = buf + "\nServer settings:\n"
	for _, item := range moderator.SettingList {
		if item.Type == moderator.TYPE_STRING {
			if *item.SData != "" || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, *item.SData)
			}
		} else if item.Type == moderator.TYPE_INT {
			if (*item.IData != 0 && *item.IData != item.DefInt) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, *item.IData)
			}
		} else if item.Type == moderator.TYPE_BOOL {
			if *item.BData != item.DefBool || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, support.BoolToString(*item.BData))
			}
		} else if item.Type == moderator.TYPE_F32 {
			if (*item.FData32 != 0 && *item.FData32 != item.DefF32) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, *item.FData32)
			}
		} else if item.Type == moderator.TYPE_F64 {
			if (*item.FData64 != 0 && *item.FData64 != item.DefF64) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, *item.FData64)
			}
		}
	}
	modStr := fact.ModLoadString
	if modStr != constants.Unknown {
		buf = buf + "\nLoaded mods: " + modStr + "\n"
	}

	/*************************
	 * Tick history
	 *************************/
	fact.TickHistoryLock.Lock()
	var tenMin []fact.TickInt
	var thirtyMin []fact.TickInt
	var oneHour []fact.TickInt

	tickHistoryLen := len(fact.TickHistory) - 1
	var tenMinAvr, thirtyMinAvr, oneHourAvr float64
	if tickHistoryLen > 0 {
		end := fact.TickHistory[tickHistoryLen]
		endInt := float64(end.Day*86400.0 + end.Hour*3600.0 + end.Min*60.0 + end.Sec)

		if tickHistoryLen >= 600 {
			tenMin = fact.TickHistory[tickHistoryLen-600 : tickHistoryLen]

			for _, item := range tenMin {
				tenMinAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}
		if tickHistoryLen >= 1800 {
			thirtyMin = fact.TickHistory[tickHistoryLen-1800.0 : tickHistoryLen]

			for _, item := range thirtyMin {
				thirtyMinAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}
		if tickHistoryLen >= 3600 {
			oneHour = fact.TickHistory[tickHistoryLen-3600 : tickHistoryLen]

			for _, item := range oneHour {
				oneHourAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}

		tenMinAvr = tenMinAvr / 180300.0 * 60.0
		thirtyMinAvr = thirtyMinAvr / 1620900.0 * 60.0
		oneHourAvr = oneHourAvr / 6481800.0 * 60.0
	}
	fact.TickHistoryLock.Unlock()

	if cfg.Local.Options.SoftModOptions.SlowConnect.Enabled && cfg.Local.Options.SoftModOptions.SlowConnect.Speed != 1.0 {
		buf = buf + fmt.Sprintf("\nUPS is set to: %2.2f\n", cfg.Local.Options.SoftModOptions.SlowConnect.Speed*60.0)
	}
	if oneHourAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f, 30m: %2.2f, 1h: %2.2f\n", tenMinAvr, thirtyMinAvr, oneHourAvr)
	} else if thirtyMinAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f, 30m: %2.2f\n", tenMinAvr, thirtyMinAvr)
	} else if tenMinAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f\n", tenMinAvr)
	}
	/* End tick history */

	buf = buf + fmt.Sprintf("\nPlayers in database: %v\n", len(glob.PlayerList))
	buf = buf + fmt.Sprintf("Players in blacklist: %v\n", len(banlist.BanList))

	if fact.PausedTicks > 4 {
		buf = buf + "\n(Server is paused)\n"
	}

	buf = buf + "```"

	msg, isConfigured := fact.MakeSteamURL()
	if isConfigured {
		buf = buf + "Steam connect link: " + msg
	}

	disc.EphemeralResponse(s, i, "Server Info:", buf)

}

func debugStat(s *discordgo.Session, i *discordgo.InteractionCreate) {

	count := 0
	glob.PlayerSusLock.Lock()
	var buf string = "Debug:\nSusList:"
	for pname, score := range glob.PlayerSus {
		if glob.PlayerSus[pname] > 0 {
			count++
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}

	glob.PlayerSusLock.Unlock()

	glob.ChatterLock.Lock()
	buf = buf + "\nChatterList:"
	for pname, score := range glob.ChatterSpamScore {
		if glob.PlayerSus[pname] > 0 {
			count++
			buf = buf + fmt.Sprintf("%v: %v\n", pname, score)
		}
	}

	glob.ChatterLock.Unlock()

	if count == 0 {
		disc.EphemeralResponse(s, i, "No debug info at this time.", "")
	} else {
		disc.EphemeralResponse(s, i, "Debug Info:", buf)
	}

}
