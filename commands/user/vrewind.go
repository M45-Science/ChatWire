package user

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"
	"os"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

//Allow regulars to vote to rewind the map
func VoteRewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	//layoutUS := "01/02 03:04 PM MST"
	argnum := len(args)
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath

	for _, player := range glob.PlayerList {
		if player.ID != "" && player.Level == 2 && player.ID == m.Author.ID {
			//vote
		} else {
			fact.CMS(m.ChannelID, "You must be `REGISTERED`, AND a `REGULAR` to use this command.")
			return
		}
	}
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
					fact.CMS(m.ChannelID, "Not Yet Implented (in-progress)")
				}
			}
		}
	}

}
