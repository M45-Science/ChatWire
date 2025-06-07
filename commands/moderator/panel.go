package moderator

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
)

// WebPanelLink sends a temporary access link to the requester.
func WebPanelLink(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if !(disc.CheckModerator(i) || disc.CheckAdmin(i)) {
		disc.InteractionEphemeralResponse(i, "Error", "You must be a moderator to use this command.")
		return
	}

	token := glob.RandomBase64String(20)
	glob.PanelTokenLock.Lock()
	glob.PanelTokens[token] = &glob.PanelTokenData{Token: token, DiscID: i.Member.User.ID, Time: time.Now().Unix()}
	glob.PanelTokenLock.Unlock()
	link := fmt.Sprintf("https://%v:%v/panel?token=%v", cfg.Global.Paths.URLs.Domain, cfg.Local.Port+constants.PanelPortOffset, token)
	disc.InteractionEphemeralResponse(i, "Panel Link", link)
}
