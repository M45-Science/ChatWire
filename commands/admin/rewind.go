package admin

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

//Load a different save-game
func Rewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath

	//Correct number of arguments (1)
	if argnum == 1 {
		num, err := strconv.Atoi(args[0])
		//Seems to be a number
		if err == nil {
			if num > 0 || num < 9999 {
				//Check if file is valid and found
				autoSaveStr := fmt.Sprintf("_autosave%v.zip", num)
				_, err := os.Stat(path + "/" + autoSaveStr)
				notfound := os.IsNotExist(err)

				if notfound {
					fact.CMS(m.ChannelID, "I don't see that autosave.")
					return
				} else {
					fact.SetAutoStart(false)
					fact.QuitFactorio()

					for x := 0; x < constants.MaxFactorioCloseWait && fact.IsFactRunning(); x++ {
						time.Sleep(time.Second)
						if x == (constants.MaxFactorioCloseWait - 1) {
							fact.CMS(m.ChannelID, "Factorio may be frozen, canceling rewind.")
							return
						}
					}
					selSaveName := path + "/" + autoSaveStr
					from, erra := os.Open(selSaveName)
					if erra != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to open the selected rewind map. Details: %s", erra))
					}
					defer from.Close()

					newmappath := path + "/" + cfg.Local.Name + "_new.zip"
					_, err := os.Stat(newmappath)
					if !os.IsNotExist(err) {
						err = os.Remove(newmappath)
						if err != nil {
							botlog.DoLog(fmt.Sprintf("An error occurred when attempting to remove the new map. Details: %s", err))
							return
						}
					}
					to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
					if errb != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to create the new rewind map. Details: %s", errb))
						return
					}
					defer to.Close()

					_, errc := io.Copy(to, from)
					if errc != nil {
						botlog.DoLog(fmt.Sprintf("An error occurred when attempting to write the new rewind map. Details: %s", errc))
						return
					}

					fact.CMS(m.ChannelID, fmt.Sprintf("Loading autosave%v", num))
					time.Sleep(time.Second * 1)
					fact.SetAutoStart(true)
					return
				}
			}
		}
	} else {
		fact.ShowRewindList(s, m)
		return
	}

	fact.CMS(m.ChannelID, "Not a valid autosave number, `"+cfg.Global.DiscordCommandPrefix+"rewind` to see a list of autosaves.")
}
