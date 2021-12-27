package user

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func ShowSettings(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := "```"
	/* STATS */
	buf = buf + "Stats:\n"
	buf = buf + fmt.Sprintf("%25v: %v-%v\n", "Server name", cfg.Local.ServerCallsign, cfg.Local.Name)
	buf = buf + fmt.Sprintf("%25v: m45sci.xyz:%v\n", "Address:", cfg.Local.Port)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire version", constants.Version)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio version", glob.FactorioVersion)
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire up-time", tnow.Sub(glob.Uptime.Round(time.Second)).String())
	if !glob.FactorioBootedAt.IsZero() && fact.IsFactorioBooted() {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio up-time", tnow.Sub(glob.FactorioBootedAt.Round(time.Second)).String())
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
	if glob.LastSaveName != constants.Unknown {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Save name", glob.LastSaveName)
	}
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map time", glob.GametimeString)
	buf = buf + fmt.Sprintf("%25v: %v (most ever %v)\n", "Players online", fact.GetNumPlayers(), glob.RecordPlayers)
	//buf = buf + fmt.Sprintf("%25v: %v\n", "Members", nummember)
	//buf = buf + fmt.Sprintf("%25v: %v\n", "Regulars", numregulars)
	//buf = buf + fmt.Sprintf("%25v: %v\n", "Registered", numreg)

	/* SETTINGS */
	buf = buf + "\nSettings:\n"
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map preset", cfg.Local.MapPreset)
	if cfg.Local.MapGenPreset != "" {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map gen", cfg.Local.MapGenPreset)
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map gen", "not set")
	}
	if !cfg.Local.AutoStart {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-start", support.BoolToString(cfg.Local.AutoStart))
	}
	if !cfg.Local.AutoUpdate {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-update", support.BoolToString(cfg.Local.AutoUpdate))
	}
	if cfg.Local.UpdateFactExp {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Experimental-updates", cfg.Local.UpdateFactExp)
	}
	if cfg.Local.ResetScheduleText != "" {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Reset schedule", cfg.Local.ResetScheduleText)
	}
	if cfg.Local.DisableBlueprints {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Disable blueprints", support.BoolToString(cfg.Local.DisableBlueprints))
	}
	if cfg.Local.EnableCheats {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Enable cheats", support.BoolToString(cfg.Local.EnableCheats))
	}
	if cfg.Local.HideAutosaves {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Hide autosaves", support.BoolToString(cfg.Local.HideAutosaves))
	}
	if cfg.Local.SlowConnect.SlowConnect {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Slow connect", support.BoolToString(cfg.Local.SlowConnect.SlowConnect))
	}

	//Softmod
	if cfg.Local.SoftModOptions.CleanMapOnBoot {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Clean map on boot", support.BoolToString(cfg.Local.SoftModOptions.CleanMapOnBoot))
	}
	if cfg.Local.SoftModOptions.DoWhitelist {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Members Only", support.BoolToString(cfg.Local.SoftModOptions.DoWhitelist))
	}
	if cfg.Local.SoftModOptions.FriendlyFire {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Friendly Fire", support.BoolToString(cfg.Local.SoftModOptions.FriendlyFire))
	}
	if !cfg.Local.SoftModOptions.RestrictMode {
		buf = buf + fmt.Sprintf("%25v: %v\n", "New Player Restrictions", support.BoolToString(cfg.Local.SoftModOptions.RestrictMode))
	}
	if !cfg.Local.FactorioData.Autopause {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Auto Pause", support.BoolToString(cfg.Local.FactorioData.Autopause))
	}
	if cfg.Local.FactorioData.Autosave_interval < 10 || cfg.Local.FactorioData.Autosave_interval > 15 {
		buf = buf + fmt.Sprintf("%25v: %vm\n", "Autosaves", cfg.Local.FactorioData.Autosave_interval)
	}

	if glob.ModLoadString != constants.Unknown {
		buf = buf + "\nMod list: " + glob.ModLoadString + "\n"
	}

	buf = buf + "\n```"
	fact.CMS(m.ChannelID, buf)

}
