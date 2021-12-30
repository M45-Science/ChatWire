package fact

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"

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
		botlog.DoLog(fmt.Sprintf("Unable to delete old sav-*/gen-* map saves. Details:\nout: %v\nerr: %v", string(out), errs))
	} else {
		botlog.DoLog(fmt.Sprintf("Deleted old sav-*/gen-* map saves. Details:\nout: %v\nerr: %v", string(out), errs))
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

func WriteWhitelist() int {

	wpath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix +
		cfg.Local.ServerCallsign + "/" + constants.WhitelistName

	if cfg.Local.SoftModOptions.DoWhitelist {
		glob.PlayerListLock.RLock()
		var count = 0
		var buf = "[\n"
		for _, player := range glob.PlayerList {
			if player.Level > 0 {
				buf = buf + "\"" + player.Name + "\",\n"
				count = count + 1
			}
		}
		lchar := len(buf)
		buf = buf[0 : lchar-2]
		buf = buf + "\n]\n"
		glob.PlayerListLock.RUnlock()

		_, err := os.Create(wpath)

		if err != nil {
			botlog.DoLog("WriteWhitelist: os.Create failure")
			return -1
		}

		err = ioutil.WriteFile(wpath, []byte(buf), 0644)

		if err != nil {
			botlog.DoLog("WriteWhitelist: WriteFile failure")
			return -1
		}
		return count
	} else {
		_ = os.Remove(wpath)
	}
	return 0
}

func QuitFactorio() {

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
	}
}

//Disabled
func SaveFactorio() {

	if IsFactorioBooted() && 1 == 2 {
		gtime := GetGameTime()

		if gtime != constants.Unknown {
			WriteFact(fmt.Sprintf("/server-save sav-%s", gtime))
		} else {
			WriteFact("/server-save")
		}
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
			botlog.DoLog("Message to factorio, too long... Not sending.")
			return
		} else if plen <= 1 {
			botlog.DoLog("Message for factorio too short... Not sending.")
			return
		}

		_, err := io.WriteString(gpipe, buf+"\n")
		if err != nil {
			botlog.DoLog(fmt.Sprintf("An error occurred when attempting to write to Factorio. Details: %s", err))
			SetFactRunning(false, true)
			return
		}

	} else {
		botlog.DoLog("An error occurred when attempting to write to Factorio (nil pipe)")
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

			WriteFact(fmt.Sprintf("/ban %s previously banned", pname))
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
			if plevel == 1 {
				newrole = cfg.Global.RoleData.MemberRoleName
			} else if plevel == 2 {
				newrole = cfg.Global.RoleData.RegularRoleName
			} else if plevel == 255 {
				newrole = cfg.Global.RoleData.AdminRoleName
			} else {
				newrole = cfg.Global.RoleData.NewRoleName
			}

			guild := GetGuild()

			if guild != nil && glob.DS != nil {

				errrole, regrole := disc.RoleExists(guild, newrole)

				if errrole {
					errset := disc.SmartRoleAdd(cfg.Global.DiscordData.GuildID, discid, regrole.ID)
					if errset != nil {
						botlog.DoLog(fmt.Sprintf("Couldn't set role %v for %v.", plevel, pname))
					}
				}
			} else {

				botlog.DoLog("No guild data.")
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

	if chname != oldchname && cfg.Local.ChannelData.ChatID != "" {
		glob.UpdateChannelLock.Lock()
		glob.OldChanName = glob.NewChanName
		glob.UpdateChannelLock.Unlock()

		_, aerr := glob.DS.ChannelEditComplex(cfg.Local.ChannelData.ChatID, &discordgo.ChannelEdit{Name: chname, Position: cfg.Local.ChannelData.Pos})

		if aerr != nil {
			botlog.DoLog(fmt.Sprintf("An error occurred when attempting to rename the Factorio discord channel. Details: %s", aerr))
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

	red := color.R + 0.2
	green := color.G + 0.2
	blue := color.B + 0.2

	if red > 1 {
		red = 1
	}
	if green > 1 {
		green = 1
	}
	if blue > 1 {
		blue = 1
	}

	if justnumbers {
		buf = fmt.Sprintf("%.2f,%.2f,%.2f", red, green, blue)
	} else {
		buf = fmt.Sprintf("[color=%.2f,%.2f,%.2f]", red, green, blue)
	}
	return buf
}
