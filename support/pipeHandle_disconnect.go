package support

import (
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

func handleDisconnect(input *handleData) bool {

	if strings.HasPrefix(input.noTimecode, "Info ServerMultiplayerManager") {

		if glob.SoftModVersion == constants.Unknown {
			if strings.Contains(input.line, "removing peer") {
				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "A player has disconnected.")
				fact.WriteFact(glob.OnlineCommand)
			}
		}
	}

	return false
}
