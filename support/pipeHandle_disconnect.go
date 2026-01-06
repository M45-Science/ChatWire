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

		if strings.Contains(input.line, "removing peer") {
			if glob.SoftModVersion == constants.Unknown {
				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "A player has disconnected.")
			}
			// Always refresh player count on disconnect events so channel names
			// and reboot-when-empty logic don't get stuck with stale player data.
			fact.WriteFact(glob.OnlineCommand)
		}
	}

	return false
}
