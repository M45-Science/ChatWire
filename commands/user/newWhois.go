package user

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"

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
	layoutUS := "01-02-06 3:04 PM"

	a := i.ApplicationCommandData()

	maxresults := constants.WhoisResults

	/* Reconstruct list, to remove empty entries and to reduce lock time */
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		slist = append(slist, *player)
	}
	glob.PlayerListLock.RUnlock()

	units, err := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")
	if err != nil {
		panic(err)
	}

	buf := ""
	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()

			/*STANDARD WHOIS SEARCH*/
			count := 0
			format := "`%20v : %10v : %20v : %18v : %18v : %10v : %10v : %v`\n"
			buf = buf + fmt.Sprintf(format, "Factorio Name", "Time", "Discord Name", "Last Seen", "Joined", "Level", "SusScore", "Ban reason")
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
					n, _ := durafmt.ParseString(fmt.Sprintf("%vm", p.Minutes))
					timestr := n.LimitFirstN(2).Format(units)
					buf = buf + fmt.Sprintf(format,
						sclean.TruncateStringEllipsis(p.Name, 20),
						timestr,
						sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID), 20),
						lseen,
						joined,
						fact.LevelToString(p.Level),
						p.SusScore,
						p.BanReason)
					count++
				}
			}
			if count == 0 {
				buf = "No results."
			}

			//1 << 6 is ephemeral/private
			respData := &discordgo.InteractionResponseData{Content: buf, Flags: 1 << 6}
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
