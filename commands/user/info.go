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
	"ChatWire/util"
)

/**************************
 * Show useful info about a server and it's settings
 *************************/
func Info(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	verbose := false

	buf := "```"

	a := i.ApplicationCommandData()
	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			str := arg.StringValue()
			if strings.EqualFold(str, "verbose") {
				verbose = true
			}
		}
	}

	/* STATS */
	if verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", constants.ProgName+" version", constants.Version)
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

	if cfg.Local.Options.PlayHourEnable {
		buf = buf + fmt.Sprintf("Time restrictions: %v - %v GMT.\n",
			cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour)
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

	if fact.HasResetTime() {
		buf = buf + fmt.Sprintf("\n%17v: %v\n", "Next map reset", fact.FormatResetTime())
		buf = buf + fmt.Sprintf("%17v: %v\n", "Time till reset", fact.TimeTillReset())
		buf = buf + fmt.Sprintf("%17v: %v\n", "Interval", fact.FormatResetInterval())
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
				buf = buf + fmt.Sprintf("%23v: %v\n", item.ShortDesc, util.BoolToOnOff(*item.BData))
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

	/*************************
	 * Tick history
	 *************************/

	tenMinAvr, thirtyMinAvr, oneHourAvr := fact.GetFactUPS()

	if oneHourAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f, 30m: %2.2f, 1h: %2.2f\n", tenMinAvr, thirtyMinAvr, oneHourAvr)
	} else if thirtyMinAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f, 30m: %2.2f\n", tenMinAvr, thirtyMinAvr)
	} else if tenMinAvr > 0 {
		buf = buf + fmt.Sprintf("UPS Average: 10m: %2.2f\n", tenMinAvr)
	}
	/* End tick history */

	glob.PlayerListLock.RLock()
	var mem, reg, vet, mod, ban int
	for _, player := range glob.PlayerList {
		switch player.Level {
		case -1:
			ban++
		case 1:
			mem++
		case 2:
			reg++
		case 3:
			vet++
		case 255:
			mod++
		}
	}
	bCount := len(banlist.BanList)
	ban += bCount

	total := ban + mem + reg + vet + mod
	buf = buf + fmt.Sprintf("Members: %v, Regulars: %v, Veterans: %v\nModerators: %v, Banned: %v, Total: %v\n",
		mem, reg, vet, mod, ban, total)
	glob.PlayerListLock.RUnlock()

	if fact.PausedTicks > 4 {
		buf = buf + "\n(Server is paused)\n"
	}

	buf = buf + "```"

	msg, isConfigured := fact.MakeSteamURL()
	if isConfigured {
		buf = buf + "Steam connect link:\n" + msg
	}

	if fact.HasResetTime() {
		buf = buf + fmt.Sprintf("\nNEXT MAP RESET: <t:%v:F>(local time)\n", cfg.Local.Options.NextReset.UTC().Unix())
	}

	disc.InteractionEphemeralResponse(i, "Server Info:", buf)
}
