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
	glob.PanelTokens[token] = &glob.PanelTokenData{Token: token, Name: i.Member.User.Username, DiscID: i.Member.User.ID, Time: time.Now().Unix()}
	glob.PanelTokenLock.Unlock()
	dom := cfg.Global.Paths.URLs.Domain
	if glob.LocalTestMode != nil && *glob.LocalTestMode {
		dom = "127.0.0.1"
	}
	link := fmt.Sprintf("https://%v:%v/panel?token=%v", dom, cfg.Local.Port+constants.PanelPortOffset, token)

	embed := &discordgo.MessageEmbed{
		Title:       "Click here to open the moderator panel",
		URL:         link,
		Description: "Run the /web-panel command again to invalidate your token if it becomes public.",
		Color:       glob.COLOR_WHITE,
	}
	embeds := []*discordgo.MessageEmbed{embed}
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: embeds, Flags: discordgo.MessageFlagsEphemeral},
	}
	err := disc.DS.InteractionRespond(i.Interaction, resp)
	if err != nil {
		newResp := &discordgo.WebhookEdit{Embeds: &embeds}
		disc.DS.InteractionResponseEdit(i.Interaction, newResp)
	}
}
