package user

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"

	"github.com/bwmarrin/discordgo"
)

//AccessServer locks PasswordListLock
func Info(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

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

	buf := "```"
	buf = buf + fmt.Sprintf("ChatWire    : %v\n", constants.Version)
	buf = buf + fmt.Sprintf("Factorio    : %v\n", glob.FactorioVersion)
	buf = buf + fmt.Sprintf("Last Reboot : %v\n", tnow.Sub(glob.Uptime.Round(time.Second)).String())
	buf = buf + fmt.Sprintf("Map Time    : %v\n", glob.GametimeString)
	buf = buf + fmt.Sprintf("Players     : %v (Record %v)\n", fact.GetNumPlayers(), glob.RecordPlayers)
	buf = buf + fmt.Sprintf("Members     : %v\n", nummember)
	buf = buf + fmt.Sprintf("Regulars    : %v\n", numregulars)
	buf = buf + fmt.Sprintf("Registered  : %v\n", numreg)

	if cfg.Local.MapGenPreset != "" {
		buf = buf + fmt.Sprintf("Map-Gen Preset : %v\n", cfg.Local.MapGenPreset)
	}
	if cfg.Local.ResetScheduleText != "" {
		buf = buf + fmt.Sprintf("Reset Schedule  : %v\n", cfg.Local.ResetScheduleText)
	}
	buf = buf + fmt.Sprintf("Slow Connect    : %v\n", support.BoolToString(cfg.Local.SlowConnect.SlowConnect))
	buf = buf + "```"
	fact.CMS(m.ChannelID, buf)

}
