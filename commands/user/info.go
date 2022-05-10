package user

import (
	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands/admin"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

/**************************
 * Show useful info about a server and it's settings
 *************************/
func ShowSettings(s *discordgo.Session, i *discordgo.InteractionCreate) {

	verbose := false

	buf := "```"
	/* STATS */
	buf = buf + fmt.Sprintf("%17v: %v\n", "ChatWire version", constants.Version)
	buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio version", fact.FactorioVersion)
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	buf = buf + fmt.Sprintf("%17v: %v\n", "ChatWire up-time", tnow.Sub(glob.Uptime.Round(time.Second)).String())
	if !fact.FactorioBootedAt.IsZero() && fact.IsFactorioBooted() {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio up-time", tnow.Sub(fact.FactorioBootedAt.Round(time.Second)).String())
	} else {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Factorio up-time", "not running")
	}

	if fact.LastSaveName != constants.Unknown || verbose {
		buf = buf + fmt.Sprintf("%17v: %v\n", "Save name", fact.LastSaveName)
	}
	buf = buf + fmt.Sprintf("%17v: %v\n", "Map time", fact.GametimeString)
	buf = buf + fmt.Sprintf("%17v: %v (most ever %v)\n", "Players online", fact.GetNumPlayers(), glob.RecordPlayers)

	/* SETTINGS */
	buf = buf + "\nServer settings:\n"
	for _, item := range admin.SettingList {
		if item.Type == admin.TYPE_STRING {
			if *item.SData != "" || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.Desc, *item.SData)
			}
		} else if item.Type == admin.TYPE_INT {
			if (*item.IData != 0 && *item.IData != item.DefInt) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.Desc, *item.IData)
			}
		} else if item.Type == admin.TYPE_BOOL {
			if *item.BData != item.DefBool || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.Desc, support.BoolToString(*item.BData))
			}
		} else if item.Type == admin.TYPE_F32 {
			if (*item.FData32 != 0 && *item.FData32 != item.DefF32) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.Desc, *item.FData32)
			}
		} else if item.Type == admin.TYPE_F64 {
			if (*item.FData64 != 0 && *item.FData64 != item.DefF64) || verbose {
				buf = buf + fmt.Sprintf("%23v: %v\n", item.Desc, *item.FData64)
			}
		}
	}
	modStr := fact.GetModLoadString()
	if modStr != "" {
		buf = buf + "\nLoaded mods: " + modStr + "\n"
	}
	banlist.BanListLock.Lock()
	banCount := len(banlist.BanList)
	banlist.BanListLock.Unlock()

	if banCount > 0 {
		buf = buf + "\nBan count " + fmt.Sprintf("%v", banCount) + "\n"
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

	if fact.GetPausedTicks() > 4 {
		buf = buf + "(Server is paused)\n"
	}

	buf = buf + "```"

	respData := &discordgo.InteractionResponseData{Content: buf}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	s.InteractionRespond(i.Interaction, resp)
	//fact.CMS(i.ChannelID, buf)

}
