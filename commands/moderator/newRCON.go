package moderator

import (
	"fmt"
	"strings"

	"github.com/Distortions81/rcon"
	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

/* Set a player's level */
func RCONCmd(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

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
			msg := "RCON was unable to connect to Factorio."
			cwlog.DoLogCW(msg)
			disc.InteractionEphemeralResponse(i, "Error:", msg)
			return
		}

		cwlog.DoLogCW(i.Member.User.Username + ": " + server + ": " + command)
		reqID, err := remoteConsole.Write(command)
		if err != nil {
			msg := "Was unable to write to RCON."
			cwlog.DoLogCW(msg)
			disc.InteractionEphemeralResponse(i, "Error:", msg)
			return
		}
		resp, respReqID, err := remoteConsole.Read()
		if err != nil {
			msg := "Was unable to read from RCON."
			cwlog.DoLogCW(msg)
			disc.InteractionEphemeralResponse(i, "Error:", msg)
			return
		}

		if reqID != respReqID {
			msg := "RCON responded with an invalid ID."
			cwlog.DoLogCW(msg)
			disc.InteractionEphemeralResponse(i, "Error:", msg)
			return
		}

		if resp == "" {
			resp = "(Empty response)"
		}
		disc.InteractionEphemeralResponse(i, "Result:", resp)
		cwlog.DoLogCW("RCON: " + resp)
	} else {
		msg := "You must supply a command to run."
		cwlog.DoLogCW(msg)
		disc.InteractionEphemeralResponse(i, "Error:", msg)
	}

}
