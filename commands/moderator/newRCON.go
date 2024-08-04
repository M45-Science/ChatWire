package moderator

import (
	"fmt"
	"log"
	"strings"

	"github.com/Distortions81/rcon"
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

/* Set a player's level */
func RCONCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var server, command string

	a := i.ApplicationCommandData()

	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			if strings.EqualFold(arg.Name, "command") {
				command = arg.StringValue()
				//Fix missing slash
				command = strings.TrimPrefix(command, "/")
				command = "/" + command
			}
		}
	}

	if command != "" {

		portstr := fmt.Sprintf("%v", cfg.Local.Port+cfg.Global.Options.RconOffset)
		remoteConsole, err := rcon.Dial("localhost"+":"+portstr, glob.RCONPass)

		if err != nil || remoteConsole == nil {
			cwlog.DoLogCW(fmt.Sprintf("Error: `%v`\n", err))

			disc.EphemeralResponse(s, i, "Error:", err.Error())
		}

		cwlog.DoLogCW(i.Member.User.Username + ": " + server + ": " + command)
		reqID, err := remoteConsole.Write(command)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		resp, respReqID, err := remoteConsole.Read()
		if err != nil {
			log.Println(err.Error())
			return
		}

		if reqID != respReqID {
			log.Println("Invalid response ID.")
			return
		}

		disc.EphemeralResponse(s, i, "Result:", resp)
		cwlog.DoLogCW("RCON RESPONSE: " + resp)
	} else {
		buf := "You must supply a command to run."
		disc.EphemeralResponse(s, i, "Error:", buf)
	}

}
