package fact

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"
)

/* Delete old sav-*.zip, gen-*.zip files, to save space. */
func DeleteOldSav() {
	patha := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath + "/sav-*.zip"
	pathb := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath + "/gen-*.zip"

	var tempargs []string
	tempargs = append(tempargs, "-f")
	tempargs = append(tempargs, patha)
	tempargs = append(tempargs, pathb)

	out, errs := exec.Command(cfg.Global.PathData.RMPath, tempargs...).Output()

	if errs != nil {
		cwlog.DoLogCW(fmt.Sprintf("Unable to delete old sav-*/gen-* map saves. Details:\nout: %v\nerr: %v", string(out), errs))
	} else {
		cwlog.DoLogCW(fmt.Sprintf("Deleted old sav-*/gen-* map saves. Details:\nout: %v\nerr: %v", string(out), errs))
	}
}

/* Whitelist a specifc player. */
func WhitelistPlayer(pname string, level int) {
	if IsFactRunning() {
		if cfg.Local.DoWhitelist {
			if level > 0 {
				WriteFact(fmt.Sprintf("/whitelist add %s", pname))
			}
		}
	}
}

/* Write a full whitelist for a server, before it boots */
func WriteWhitelist() int {

	wpath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix +
		cfg.Local.ServerCallsign + "/" + constants.WhitelistName

	if cfg.Local.DoWhitelist {
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
			cwlog.DoLogCW("WriteWhitelist: os.Create failure")
			return -1
		}

		err = ioutil.WriteFile(wpath, []byte(buf), 0644)

		if err != nil {
			cwlog.DoLogCW("WriteWhitelist: WriteFile failure")
			return -1
		}
		return count
	} else {
		_ = os.Remove(wpath)
	}

	time.Sleep(time.Millisecond * 100)
	return 0
}

 /* Quit Factorio */
func QuitFactorio() {

	SetRelaunchThrottle(0)
	SetNoResponseCount(0)

	/* Running but no players, just quit */
	if IsFactorioBooted() && GetNumPlayers() <= 0 {
		WriteFact("/quit")

		/* Running, but players connected... Give them quick feedback. */
	} else if IsFactorioBooted() && GetNumPlayers() > 0 {
		WriteFact("/cchat [color=red]Server quitting.[/color]")
		WriteFact("/cchat [color=green]Server quitting..[/color]")
		WriteFact("/cchat [color=blue]Server quitting...[/color]")
		WriteFact("/quit")
	}
}

/* Tell Factorio to save the map */
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

/* Send a string to Factorio, via stdin */
func WriteFact(input string) {
	PipeLock.Lock()
	defer PipeLock.Unlock()

	/* Clean string */
	buf := sclean.StripControlAndSubSpecial(input)

	gpipe := Pipe
	if gpipe != nil {

		plen := len(buf)

		if plen > 2000 {
			cwlog.DoLogCW("Message to Factorio, too long... Not sending.")
			return
		} else if plen <= 1 {
			cwlog.DoLogCW("Message for Factorio too short... Not sending.")
			return
		}

		_, err := io.WriteString(gpipe, buf+"\n")
		if err != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to write to Factorio. Details: %s", err))
			SetFactRunning(false, true)
			return
		}

	} else {
		cwlog.DoLogCW("An error occurred when attempting to write to Factorio (nil pipe)")
		SetFactRunning(false, true)
		return
	}
}

func LevelToString(level int) string {

	name := "Invalid"

	if level <= -254 {
		name = "Deleted"
	} else if level == -1 {
		name = "Banned"
	} else if level == 0 {
		name = "New"
	} else if level == 1 {
		name = "Member"
	} else if level == 2 {
		name = "Regular"
	} else if level >= 255 {
		name = "Admin"
	}

	return name
}

/* Promote a player to the level they have, in Factorio and on Discord */
func AutoPromote(pname string) string {
	playerName := " *(New Player)* "

	if pname != "" {
		plevel := PlayerLevelGet(pname, false)
		if plevel <= -254 {
			playerName = " **(Deleted Player)** "

		} else if plevel == -1 {
			playerName = " **(Banned)**"
			WriteFact(fmt.Sprintf("/ban %s (previously banned)", pname))

		} else if plevel == 1 {
			playerName = " *(Member)*"
			WriteFact(fmt.Sprintf("/member %s", pname))

		} else if plevel == 2 {
			playerName = " *(Regular)*"

			WriteFact(fmt.Sprintf("/regular %s", pname))
		} else if plevel == 255 {
			playerName = " *(Moderator)*"

			WriteFact(fmt.Sprintf("/promote %s", pname))
		}

		discid := disc.GetDiscordIDFromFactorioName(pname)
		factname := disc.GetFactorioNameFromDiscordID(discid)

		if factname == pname {

			newrole := ""
			if plevel == 0 {
				newrole = cfg.Global.RoleData.NewRoleName
			} else if plevel == 1 {
				newrole = cfg.Global.RoleData.MemberRoleName
			} else if plevel == 2 {
				newrole = cfg.Global.RoleData.RegularRoleName
			} else if plevel == 255 {
				newrole = cfg.Global.RoleData.ModeratorRoleName
			}

			guild := GetGuild()

			if guild != nil && disc.DS != nil {

				errrole, regrole := disc.RoleExists(guild, newrole)

				if errrole {
					errset := disc.SmartRoleAdd(cfg.Global.DiscordData.GuildID, discid, regrole.ID)
					if errset != nil {
						cwlog.DoLogCW(fmt.Sprintf("Couldn't set role %v for %v.", newrole, discid))
					}
				}
			} else {

				cwlog.DoLogCW("No guild data.")
			}
		}
	}

	return playerName

}

/* Update our channel name, but don't send it yet */
func UpdateChannelName() {

	var newchname string
	nump := GetNumPlayers()

	if nump == 0 {
		newchname = fmt.Sprintf("%v", cfg.Local.ServerCallsign+"-"+cfg.Local.Name)
	} else {
		newchname = fmt.Sprintf("%v🔵%v", nump, cfg.Local.ServerCallsign+"-"+cfg.Local.Name)
	}

	disc.UpdateChannelLock.Lock()
	disc.NewChanName = newchname
	disc.UpdateChannelLock.Unlock()

}

/* When appropriate, actually update the channel name */
func DoUpdateChannelName() {

	if disc.DS == nil {
		return
	}

	disc.UpdateChannelLock.Lock()
	chname := disc.NewChanName
	oldchname := disc.OldChanName
	disc.UpdateChannelLock.Unlock()

	if chname != oldchname && cfg.Local.ChannelData.ChatID != "" {
		disc.UpdateChannelLock.Lock()
		disc.OldChanName = disc.NewChanName
		disc.UpdateChannelLock.Unlock()

		_, aerr := disc.DS.ChannelEditComplex(cfg.Local.ChannelData.ChatID, &discordgo.ChannelEdit{Name: chname, Position: cfg.Local.ChannelData.Pos})

		if aerr != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to rename the Factorio discord channel. Details: %s", aerr))
		}
	}
}

func ShowRewindList(s *discordgo.Session, m *discordgo.MessageCreate) {
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath

	files, err := ioutil.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		log.Fatal(err)
		CMS(m.ChannelID, "Error: Unable to read autosave directory.")
	}

	lastNum := -1
	step := 1
	/* Loop all files */
	var tempf []fs.FileInfo
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "_autosave") && strings.HasSuffix(f.Name(), ".zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	maxList := constants.MaxRewindResults
	buf := "Last 40 autosaves:\n"

	numFiles := len(tempf) - 1
	startPos := 0
	if numFiles > maxList {
		startPos = maxList
	} else {
		startPos = numFiles
	}

	for i := startPos; i > 0; i-- {

		f := tempf[i]
		fName := f.Name()

		/* Check if its a properly name autosave */
		if strings.HasPrefix(fName, "_autosave") && strings.HasSuffix(fName, ".zip") {
			fTmp := strings.TrimPrefix(fName, "_autosave")
			fNumStr := strings.TrimSuffix(fTmp, ".zip")
			fNum, err := strconv.Atoi(fNumStr) /* autosave number
			/* Nope, no valid number */
			if err != nil {
				continue
			}
			step++

			units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
			if err != nil {
				panic(err)
			}

			/* Get mod date */
			modDate := time.Since(f.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(3).Format(units)
			/* Add to list with mod date */
			buf = buf + fmt.Sprintf("`#%-3v: %-20v`\n", fNum, modStr+" ago")
			lastNum = fNum
		}
	}

	if lastNum == -1 {
		CMS(m.ChannelID, "No autosaves found.")
	} else {
		CMS(m.ChannelID, buf)
	}
}

func DoRewindMap(s *discordgo.Session, m *discordgo.MessageCreate, arg string) {
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath
	num, err := strconv.Atoi(arg)
	/* Seems to be a number */
	if err == nil {
		if num > 0 || num < 9999 {
			/* Check if file is valid and found */
			autoSaveStr := fmt.Sprintf("_autosave%v.zip", num)
			_, err := os.Stat(path + "/" + autoSaveStr)
			notfound := os.IsNotExist(err)

			if notfound {
				rewindSyntax(m)
				return
			} else {
				SetAutoStart(false)
				QuitFactorio()

				for x := 0; x < constants.MaxFactorioCloseWait && IsFactRunning(); x++ {
					time.Sleep(time.Second)
					if x == (constants.MaxFactorioCloseWait - 1) {
						CMS(m.ChannelID, "Factorio may be frozen, canceling rewind.")
						return
					}
				}
				selSaveName := path + "/" + autoSaveStr
				from, erra := os.Open(selSaveName)
				if erra != nil {
					cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to open the selected rewind map. Details: %s", erra))
				}
				defer from.Close()

				newmappath := path + "/" + sclean.UnixSafeFilename(cfg.Local.Name) + "_new.zip"
				_, err := os.Stat(newmappath)
				if !os.IsNotExist(err) {
					err = os.Remove(newmappath)
					if err != nil {
						cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to remove the new map. Details: %s", err))
						return
					}
				}
				to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
				if errb != nil {
					cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the new rewind map. Details: %s", errb))
					return
				}
				defer to.Close()

				_, errc := io.Copy(to, from)
				if errc != nil {
					cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to write the new rewind map. Details: %s", errc))
					return
				}

				CMS(m.ChannelID, fmt.Sprintf("Loading autosave%v", num))
				SetAutoStart(true)
				return
			}
		} else {
			rewindSyntax(m)
		}
	} else {
		rewindSyntax(m)
	}
}

func rewindSyntax(m *discordgo.MessageCreate) {
	CMS(m.ChannelID, "Not a valid autosave number, `"+cfg.Global.DiscordCommandPrefix+"rewind` to see a list of autosaves.")
	CMS(m.ChannelID, "Syntax: `"+cfg.Global.DiscordCommandPrefix+"rewind <autosave number>`")
}
