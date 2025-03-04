package user

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

/*  Get info on a specific player */
func Whois(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	var slist []glob.PlayerData
	layoutUS := "01-02-06"

	a := i.ApplicationCommandData()

	maxresults := constants.WhoisResults

	/* Reconstruct list, to remove empty entries and to reduce lock time */
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		slist = append(slist, *player)
	}
	glob.PlayerListLock.RUnlock()

	buf := ""
	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()

			/*STANDARD WHOIS SEARCH*/
			count := 0
			format := "\n```%7v: %v\n%7v: %v\n%7v: %v\n%7v: %v\n%7v: %v\n%7v: %v\n%7v: %v\n%7v: %v```\n"
			for _, p := range slist {
				if count > maxresults {
					break
				}
				if strings.Contains(strings.ToLower(p.Name), strings.ToLower(arg)) || strings.Contains(strings.ToLower(disc.GetNameFromID(p.ID)), strings.ToLower(arg)) {

					lseen := ""
					if fact.IsPlayerOnline(p.Name) {
						lseen = "Online"
					} else {
						if p.LastSeen == 0 {
							lseen = constants.Unknown
						} else {
							ltime := fact.ExpandTime(p.LastSeen)
							lseen = ltime.Format(layoutUS)
						}
					}

					joined := ""
					if p.Creation == 0 {
						joined = constants.Unknown
					} else {
						jtime := fact.ExpandTime(p.Creation)
						joined = jtime.Format(layoutUS)
					}
					timestr := p.Minutes
					buf = buf + fmt.Sprintf(format,
						"Name",
						sclean.TruncateStringEllipsis(p.Name, 20),
						"Score",
						timestr,
						"Discord",
						sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID), 20),
						"Seen",
						lseen,
						"Joined",
						joined,
						"Level",
						fact.LevelToString(p.Level),
						"Sus",
						p.SusScore,
						"Ban",
						p.BanReason)
					count++
				}
			}
			if count == 0 {
				buf = "No results."
			}

			respData := &discordgo.InteractionResponseData{Content: buf, Flags: discordgo.MessageFlagsEphemeral}
			resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
			err := disc.DS.InteractionRespond(i.Interaction, resp)
			if err != nil {
				return
			}
			return
		}
	}
	disc.InteractionEphemeralResponse(i, "Error:", "You must supply a search term.")

}
