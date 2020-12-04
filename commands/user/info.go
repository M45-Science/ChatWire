package user

import (
	"fmt"
	"time"

	"../../constants"
	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
)

//AccessServer locks PasswordListLock
func Info(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	//locks PlayerListLock (READ)
	numreg := 0
	nummember := 0
	numregulars := 0

	glob.PlayerListLock.RLock()
	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].ID != "" {
			numreg++
		}
		if glob.PlayerList[i].Level == 1 {
			nummember++
		} else if glob.PlayerList[i].Level == 2 {
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
	buf = buf + fmt.Sprintf("Registered  : %v", numreg)
	buf = buf + "```\n"
	fact.CMS(m.ChannelID, buf)

}
