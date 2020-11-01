package fact

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"../config"
	"../constants"
	"../disc"
	"../glob"
	"../logs"

	"github.com/bwmarrin/discordgo"
)

func FactorioIsOffline(err bool) {

	if IsFactorioBooted() {
		if err {
			LogCMS(config.Config.FactorioChannelID, "Factorio encountered an error, and is now offline.")
		} else {
			LogCMS(config.Config.FactorioChannelID, "Factorio is now offline.")
		}
	}

	SetNumPlayers(0)
	SetFactorioBooted(false)
}

func WhitelistPlayer(pname string, level int) {
	if IsFactRunning() {
		if glob.WhitelistMode {
			if level > 1 {
				WriteFact(fmt.Sprintf("/whitelist add %s", pname))
			}
		}
	}
}

func QuitFactorio() {
	SetRelaunchThrottle(0)
	SetNoResponseCount(0)
	if IsFactorioBooted() && GetNumPlayers() > 0 {
		WriteFact(fmt.Sprintf("%sServer closing.[/color]", RandomColor(false)))
		WriteFact(fmt.Sprintf("%sServer closing..[/color]", RandomColor(false)))
		WriteFact(fmt.Sprintf("%sServer closing...[/color]", RandomColor(false)))
		time.Sleep(5 * time.Second)
	}
	WriteFact("/quit")
}

func SaveFactorio() {

	if IsFactorioBooted() {
		gtime := GetGameTime()

		if gtime != constants.Unknown {
			WriteFact(fmt.Sprintf("/server-save sav-%s", gtime))
		} else {
			WriteFact("/server-save")
		}

		SetSaveTimer()
	}
}

func WriteFact(buf string) {
	glob.PipeLock.Lock()
	defer glob.PipeLock.Unlock()

	running := IsFactRunning()
	if running {
		gpipe := glob.Pipe
		if gpipe != nil {

			plen := len(buf)

			if plen > 2000 {
				logs.LogWithoutEcho("Message to factorio, too long... Not sending.")
				return
			} else if plen <= 1 {
				logs.LogWithoutEcho("Message for factorio too short... Not sending.")
				return
			}

			//time.Sleep(100 * time.Millisecond)
			_, err := io.WriteString(gpipe, buf+"\n")
			if err != nil {
				logs.LogWithoutEcho(fmt.Sprintf("An error occurred when attempting to write to Factorio. Details: %s", err))
				SetFactRunning(false, true)
				return
			}

		} else {
			logs.LogWithoutEcho("An error occurred when attempting to write to Factorio (nil pipe)")
			SetFactRunning(false, true)
			return
		}
	}
}

func AutoPromote(pname string) string {
	newusername := " *(New Player)* "

	plevel := PlayerLevelGet(pname)
	if plevel == -1 {
		newusername = " *(Banned)*"

		WriteFact(fmt.Sprintf("/ban %s", pname))
	} else if plevel == 1 {
		newusername = " *(Trusted)*"

		WriteFact(fmt.Sprintf("/member %s", pname))
	} else if plevel == 2 {
		newusername = " *(Regular)*"

		WriteFact(fmt.Sprintf("/regular %s", pname))
	} else if plevel == 255 {
		newusername = " *(Admin)*"

		WriteFact(fmt.Sprintf("/promote %s", pname))
	}

	discid := disc.GetDiscordIDFromFactorioName(pname)
	factname := disc.GetFactorioNameFromDiscordID(discid)

	if factname == pname {

		newrole := ""
		if plevel == 0 {
			newrole = config.Config.MembersRole
		} else if plevel == 1 {
			newrole = config.Config.MembersRole
		} else if plevel == 2 {
			newrole = config.Config.RegularsRole
		} else if plevel == 255 {
			newrole = config.Config.AdminsRole
		}

		guild := GetGuild()

		if guild != nil && glob.DS != nil {

			errrole, regrole := disc.RoleExists(guild, newrole)

			if errrole {
				errset := disc.SmartRoleAdd(config.Config.GuildID, discid, regrole.ID)
				if errset != nil {
					logs.Log(fmt.Sprintf("Couldn't set role %v for %v.", plevel, pname))
				}
			}
		} else {

			logs.Log("No guild data.")
		}
	}

	return newusername

}

func UpdateChannelName() {

	var newchname string
	nump := GetNumPlayers()

	if nump == 0 {
		newchname = fmt.Sprintf("%v", config.Config.ChannelName)
	} else {
		newchname = fmt.Sprintf("%v🟢%v", nump, config.Config.ChannelName)
	}

	glob.UpdateChannelLock.Lock()
	glob.OldChanName = glob.NewChanName
	glob.NewChanName = newchname
	glob.UpdateChannelLock.Unlock()

}

func DoUpdateChannelName() {

	glob.UpdateChannelLock.Lock()
	defer glob.UpdateChannelLock.Unlock()

	if glob.DS == nil {
		return
	}

	chpos, _ := strconv.Atoi(config.Config.ChannelPos)

	if glob.OldChanName != glob.NewChanName {
		glob.OldChanName = glob.NewChanName

		_, aerr := glob.DS.ChannelEditComplex(config.Config.FactorioChannelID, &discordgo.ChannelEdit{Name: glob.NewChanName, Position: chpos + 200})

		if aerr != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to rename the Factorio discord channel. Details: %s", aerr))
		}
	}
}

func RandomColor(justnumbers bool) string {
	var buf string

	if glob.LastColor < (constants.NumColors - 1) {
		glob.LastColor++
	} else {
		glob.LastColor = 0
	}

	color := constants.Colors[glob.LastColor]

	red := color.R
	green := color.G
	blue := color.B

	if red > 1.0 {
		red = 1.0
	}
	if green > 1.0 {
		green = 1.0
	}
	if blue > 1.0 {
		blue = 1.0
	}

	if justnumbers {
		buf = fmt.Sprintf("%.2f,%.2f,%.2f", red, green, blue)
	} else {
		buf = fmt.Sprintf("[color=%.2f,%.2f,%.2f]", red, green, blue)
	}
	return buf
}
