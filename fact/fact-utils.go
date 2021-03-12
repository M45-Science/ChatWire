package fact

import (
	"fmt"
	"io"
	"os/exec"
	"time"

	"../cfg"
	"../constants"
	"../disc"
	"../glob"
	"../logs"
	"../sclean"
	"github.com/bwmarrin/discordgo"
)

func DeleteOldSav() {
	//Delete old sav-*.zip, gen-*.zip files, to save space.
	patha := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath + "/sav-*.zip"
	pathb := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath + "/gen-*.zip"

	var tempargs []string
	tempargs = append(tempargs, "-f")
	tempargs = append(tempargs, patha)
	tempargs = append(tempargs, pathb)

	out, errs := exec.Command(cfg.Global.PathData.RMPath, tempargs...).Output()

	if errs != nil {
		logs.Log(fmt.Sprintf("Unable to delete old sav-*/gen-* map saves. Details:\nout: %v\nerr: %v", string(out), errs))
	} else {
		logs.Log("Deleted old sav-*/gen-* map saves.")
	}
}

func FactorioIsOffline(err bool) {

	if IsFactorioBooted() {
		if err {
			LogCMS(cfg.Local.ChannelData.ChatID, "Factorio encountered an error, and is now offline.")
		} else {
			LogCMS(cfg.Local.ChannelData.ChatID, "Factorio is now offline.")
		}
	}

	SetNumPlayers(0)
	SetFactorioBooted(false)
}

func WhitelistPlayer(pname string, level int) {
	if IsFactRunning() {
		if cfg.Local.SoftModOptions.DoWhitelist {
			if level > 0 {
				WriteFact(fmt.Sprintf("/whitelist add %s", pname))
			}
		}
	}
}

func QuitFactorio() {

	timer := GetFactQuitTimer()

	//See if we have a quit timer or not, if we don't... start one.
	if timer.IsZero() {
		StartFactQuitTimer()
	} else {
		//We already have a timer going!
		return
	}

	SetRelaunchThrottle(0)
	SetNoResponseCount(0)

	//Running but no players, just quit
	if IsFactorioBooted() && GetNumPlayers() <= 0 {
		WriteFact("/quit")

		//Running, but players connected... Give them quick feedback.
	} else if IsFactorioBooted() && GetNumPlayers() > 0 {
		WriteFact(fmt.Sprintf("/cchat %sServer quitting.[/color]", RandomColor(false)))
		WriteFact(fmt.Sprintf("/cchat %sServer quitting..[/color]", RandomColor(false)))
		WriteFact(fmt.Sprintf("/cchat %sServer quitting...[/color]", RandomColor(false)))
		time.Sleep(5 * time.Second)
		WriteFact("/quit")
	} else {
		//Not running, just reboot
		DoExit()
		return
	}
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

func WriteFact(input string) {
	glob.PipeLock.Lock()
	defer glob.PipeLock.Unlock()

	//Clean string
	buf := sclean.StripControlAndSubSpecial(input)

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

func AutoPromote(pname string) string {
	newusername := " *(New Player)* "

	if pname != "" {
		plevel := PlayerLevelGet(pname)
		if plevel == -1 {
			newusername = " *(Banned)*"

			WriteFact(fmt.Sprintf("/ban %s", pname))
		} else if plevel == 1 {
			newusername = " *(Member)*"

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
				newrole = cfg.Global.RoleData.Members
			} else if plevel == 1 {
				newrole = cfg.Global.RoleData.Members
			} else if plevel == 2 {
				newrole = cfg.Global.RoleData.Regulars
			} else if plevel == 255 {
				newrole = cfg.Global.RoleData.Admins
			}

			guild := GetGuild()

			if guild != nil && glob.DS != nil {

				errrole, regrole := disc.RoleExists(guild, newrole)

				if errrole {
					errset := disc.SmartRoleAdd(cfg.Global.DiscordData.GuildID, discid, regrole.ID)
					if errset != nil {
						logs.Log(fmt.Sprintf("Couldn't set role %v for %v.", plevel, pname))
					}
				}
			} else {

				logs.Log("No guild data.")
			}
		}
	}

	return newusername

}

func UpdateChannelName() {

	var newchname string
	nump := GetNumPlayers()

	if nump == 0 {
		newchname = fmt.Sprintf("%v", cfg.Local.ServerCallsign+"-"+cfg.Local.Name)
	} else {
		newchname = fmt.Sprintf("%v🟢%v", nump, cfg.Local.ServerCallsign+"-"+cfg.Local.Name)
	}

	glob.UpdateChannelLock.Lock()
	glob.NewChanName = newchname
	glob.UpdateChannelLock.Unlock()

}

func DoUpdateChannelName() {

	if glob.DS == nil {
		return
	}

	glob.UpdateChannelLock.Lock()
	chname := glob.NewChanName
	oldchname := glob.OldChanName
	glob.UpdateChannelLock.Unlock()

	if chname != oldchname {
		glob.UpdateChannelLock.Lock()
		glob.OldChanName = glob.NewChanName
		glob.UpdateChannelLock.Unlock()

		_, aerr := glob.DS.ChannelEditComplex(cfg.Local.ChannelData.ChatID, &discordgo.ChannelEdit{Name: chname, Position: cfg.Local.ChannelData.Pos})

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

	red := color.R + 0.1
	green := color.G + 0.1
	blue := color.B + 0.1

	if red > 0.9 {
		red = 0.9
	}
	if green > 0.9 {
		green = 0.9
	}
	if blue > 0.9 {
		blue = 0.9
	}

	if justnumbers {
		buf = fmt.Sprintf("%.2f,%.2f,%.2f", red, green, blue)
	} else {
		buf = fmt.Sprintf("[color=%.2f,%.2f,%.2f]", red, green, blue)
	}
	return buf
}
