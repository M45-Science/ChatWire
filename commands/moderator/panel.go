package moderator

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/panel"
)

// WebPanelLink sends a temporary access link to the requester.
func WebPanelLink(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if !(disc.CheckModerator(i) || disc.CheckAdmin(i)) {
		disc.InteractionEphemeralResponse(i, "Error", "You must be a moderator to use this command.")
		return
	}

	token := panel.GenerateToken(i.Member.User.ID)
	link := fmt.Sprintf("https://%v:%v/panel?token=%v", cfg.Global.Paths.URLs.Domain, cfg.Local.Port+constants.PanelPortOffset, token)
	disc.InteractionEphemeralResponse(i, "Panel Link", link)
}
