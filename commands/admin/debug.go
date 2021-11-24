package admin

import (
	"fmt"
	"time"

	"github.com/Distortions81/M45-ChatWire/constants"
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/Distortions81/M45-ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

//StatServer locks PlayerListLock (READ)
func Debug(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	numreg := 0
	numnew := 0
	numtrust := 0
	numregulars := 0
	numadmin := 0

	glob.PlayerListLock.RLock()
	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].ID != "" {
			numreg++
		}

		if glob.PlayerList[i].Level == 0 {
			numnew++
		} else if glob.PlayerList[i].Level == 1 {
			numtrust++
		} else if glob.PlayerList[i].Level == 2 {
			numregulars++
		} else if glob.PlayerList[i].Level == 255 {
			numadmin++
		}
	}
	glob.PlayerListLock.RUnlock()

	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	mm := fact.GetManMinutes()

	//Could use a refresh
	buf := "```"
	buf = buf + fmt.Sprintf("Game time: %v, Logins today: %v, Players Known: %v\n", fact.GetGameTime(), fact.GetNumLogins(), glob.PlayerListMax)
	buf = buf + fmt.Sprintf("Registered: %v, New: %v, Trusted: %v\n", numreg, numnew, numtrust)
	buf = buf + fmt.Sprintf("Regulars: %v, Admins: %v, Uptime: %v\n", numregulars, numadmin, tnow.Sub(glob.Uptime.Round(time.Second)).String())
	buf = buf + fmt.Sprintf("CW-Version: %v, Guild: %v, Log: %v\n", constants.Version, fact.GetGuildName(), glob.GameLogName)
	buf = buf + fmt.Sprintf("Paused: %v, Running: %v, AutoStart: %v\n", fact.GetPausedTicks(), fact.IsFactRunning(), fact.IsSetAutoStart())
	buf = buf + fmt.Sprintf("Reboot: %v, Queued: %v, Players: %v\n", fact.IsSetRebootBot(), fact.IsQueued(), fact.GetNumPlayers())
	buf = buf + fmt.Sprintf("Record Players: %v, RThrottle: %v, Booted: %v\n", glob.RecordPlayers, fact.GetRelaunchThrottle(), fact.IsFactorioBooted())
	buf = buf + fmt.Sprintf("Man-hours: %.4f, Activity index: %.4f", float64(mm)/60.0, float64(mm)/tnow.Sub(glob.Uptime.Round(time.Second)).Minutes())
	buf = buf + "```\n"
	fact.CMS(m.ChannelID, buf)

}
