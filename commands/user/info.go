package user

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

//Show useful info about a server and it's settings
func ShowSettings(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	verbose := false
	if len(args) > 0 && strings.EqualFold(args[0], "verbose") {
		verbose = true
	}

	buf := "```"
	/* STATS */
	buf = buf + "Stats:\n"
	buf = buf + fmt.Sprintf("%25v: %v-%v\n", "Server name", cfg.Local.ServerCallsign, cfg.Local.Name)
	buf = buf + fmt.Sprintf("%25v: m45sci.xyz:%v\n", "Address", cfg.Local.Port)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire version", constants.Version)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio version", fact.FactorioVersion)
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire up-time", tnow.Sub(glob.Uptime.Round(time.Second)).String())
	if !fact.FactorioBootedAt.IsZero() && fact.IsFactorioBooted() {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio up-time", tnow.Sub(fact.FactorioBootedAt.Round(time.Second)).String())
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio up-time", "not running")
	}
	numreg := 0
	nummember := 0
	numregulars := 0

	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		if player.ID != "" {
			numreg++
		}
		if player.Level == 1 {
			nummember++
		} else if player.Level == 2 {
			numregulars++
		}
	}
	glob.PlayerListLock.RUnlock()
	if fact.LastSaveName != constants.Unknown || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Save name", fact.LastSaveName)
	}
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map time", fact.GametimeString)
	buf = buf + fmt.Sprintf("%25v: %v (most ever %v)\n", "Players online", fact.GetNumPlayers(), glob.RecordPlayers)

	/* SETTINGS */
	buf = buf + "\nSettings:\n"
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map preset", cfg.Local.MapPreset)
	if cfg.Local.MapGenPreset != "" || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map gen", cfg.Local.MapGenPreset)
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map gen", "not set")
	}
	if !cfg.Local.AutoStart || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-start", support.BoolToString(cfg.Local.AutoStart))
	}
	if !cfg.Local.AutoUpdate || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-update", support.BoolToString(cfg.Local.AutoUpdate))
	}
	if cfg.Local.UpdateFactExp || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Experimental-updates", cfg.Local.UpdateFactExp)
	}
	if cfg.Local.ResetScheduleText != "" || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Reset schedule", cfg.Local.ResetScheduleText)
	}
	if cfg.Local.DisableBlueprints || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Disable blueprints", support.BoolToString(cfg.Local.DisableBlueprints))
	}
	if cfg.Local.EnableCheats || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Enable cheats", support.BoolToString(cfg.Local.EnableCheats))
	}
	if cfg.Local.HideAutosaves || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Hide autosaves", support.BoolToString(cfg.Local.HideAutosaves))
	}
	if cfg.Local.SlowConnect.SlowConnect || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Slow connect", support.BoolToString(cfg.Local.SlowConnect.SlowConnect))
	}

	//Softmod
	if cfg.Local.SoftModOptions.CleanMapOnBoot || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Clean map on boot", support.BoolToString(cfg.Local.SoftModOptions.CleanMapOnBoot))
	}
	if cfg.Local.SoftModOptions.DoWhitelist || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Members Only", support.BoolToString(cfg.Local.SoftModOptions.DoWhitelist))
	}
	if cfg.Local.SoftModOptions.FriendlyFire || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Friendly Fire", support.BoolToString(cfg.Local.SoftModOptions.FriendlyFire))
	}
	if !cfg.Local.SoftModOptions.RestrictMode || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "New Player Restrictions", support.BoolToString(cfg.Local.SoftModOptions.RestrictMode))
	}
	if !cfg.Local.FactorioData.AutoPause || verbose {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto Pause", support.BoolToString(cfg.Local.FactorioData.AutoPause))
	}
	if cfg.Local.FactorioData.AutoSaveMinutes < 10 || cfg.Local.FactorioData.AutoSaveMinutes > 15 || verbose {
		buf = buf + fmt.Sprintf("%25v: %vm\n", "Autosaves", cfg.Local.FactorioData.AutoSaveMinutes)
	}

	if fact.ModLoadString != constants.Unknown {
		buf = buf + "\nMod list: " + fact.ModLoadString + "\n"
	}

	buf = buf + "\n```"

	if !verbose {
		buf = buf + "More info: `$info verbose`\n"
	}
	fact.CMS(m.ChannelID, buf)

}
