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

	buf = buf + fmt.Sprintf("%25v: %v-%v\n", "Name", cfg.Local.ServerCallsign, cfg.Local.Name)
	buf = buf + fmt.Sprintf("%25v: m45sci.xyz:%v\n", "Address:", cfg.Local.Port)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map preset", cfg.Local.MapPreset)
	if cfg.Local.MapGenPreset != "" {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map settings", cfg.Local.MapGenPreset)
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Map settings", "not set")
	}
	buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-start", support.BoolToString(cfg.Local.AutoStart))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Auto-update", support.BoolToString(cfg.Local.AutoUpdate))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Experimental-updates", cfg.Local.UpdateFactExp)
	if cfg.Local.ResetScheduleText != "" {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Reset schedule", cfg.Local.ResetScheduleText)
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Reset schedule", "not set")
	}
	buf = buf + fmt.Sprintf("%25v: %v\n", "Disable blueprints", support.BoolToString(cfg.Local.DisableBlueprints))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Enable cheats", support.BoolToString(cfg.Local.EnableCheats))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Hide autosaves", support.BoolToString(cfg.Local.HideAutosaves))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Slow connect", support.BoolToString(cfg.Local.SlowConnect.SlowConnect))

	//Softmod
	buf = buf + fmt.Sprintf("%25v: %v\n", "Clean map on boot", support.BoolToString(cfg.Local.SoftModOptions.CleanMapOnBoot))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Members Only", support.BoolToString(cfg.Local.SoftModOptions.DoWhitelist))
	buf = buf + fmt.Sprintf("%25v: %v\n", "Friendly Fire", support.BoolToString(cfg.Local.SoftModOptions.FriendlyFire))
	buf = buf + fmt.Sprintf("%25v: %v\n", "New Player Restrictions", support.BoolToString(cfg.Local.SoftModOptions.RestrictMode))

	//locks PlayerListLock (READ)
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

	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire version", constants.Version)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio version", glob.FactorioVersion)
	buf = buf + fmt.Sprintf("%25v: %v\n", "ChatWire up-time", tnow.Sub(glob.Uptime.Round(time.Second)).String())
	if !glob.FactorioBootedAt.IsZero() && fact.IsFactorioBooted() {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio up-time", tnow.Sub(glob.FactorioBootedAt.Round(time.Second)).String())
	} else {
		buf = buf + fmt.Sprintf("%25v: %v\n", "Factorio up-time", "not running")
	}
	buf = buf + fmt.Sprintf("%25v: %v\n", "Map time", glob.GametimeString)
	buf = buf + fmt.Sprintf("%25v: %v (most ever %v)\n", "Players online", fact.GetNumPlayers(), glob.RecordPlayers)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Members", nummember)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Regulars", numregulars)
	buf = buf + fmt.Sprintf("%25v: %v\n", "Registered", numreg)

	if glob.ModLoadString == constants.Unknown {
		buf = buf + "No game mods loaded.\n"
	} else {
		buf = buf + "Mod list: " + glob.ModLoadString + "\n"
	}

	buf = buf + "\n"
	buf = buf + "https://wiki.factorio.com/Map_generator\n"
	buf = buf + "https://github.com/Distortions81/M45-MapSettings\n"

	buf = buf + "\n```"
	fact.CMS(m.ChannelID, buf)

}
