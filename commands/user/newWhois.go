package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

/* SORT FUNCTIONS */
/* Last Seen */
type ByLastSeen []glob.PlayerData

func (a ByLastSeen) Len() int           { return len(a) }
func (a ByLastSeen) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLastSeen) Less(i, j int) bool { return a[i].LastSeen > a[j].LastSeen }

/* Created time */
type ByNew []glob.PlayerData

func (a ByNew) Len() int           { return len(a) }
func (a ByNew) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNew) Less(i, j int) bool { return a[i].Creation > a[j].Creation }

/*  Get info on a specific player */
func Whois(s *discordgo.Session, i *discordgo.InteractionCreate) {

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

	buf := ""
	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()

			/*STANDARD WHOIS SEARCH*/
			count := 0
			buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")
			for _, p := range slist {
				if count > maxresults {
					break
				}
				if strings.Contains(strings.ToLower(p.Name), strings.ToLower(arg)) || strings.Contains(strings.ToLower(disc.GetNameFromID(p.ID, false)), strings.ToLower(arg)) {

					lseen := ""
					if p.LastSeen == 0 {
						lseen = constants.Unknown
					} else {
						ltime := fact.ExpandTime(p.LastSeen)
						lseen = ltime.Format(layoutUS)
					}

					joined := ""
					if p.Creation == 0 {
						joined = constants.Unknown
					} else {
						jtime := time.Unix((p.Creation+constants.CWEpoch)*60, 0)
						joined = jtime.Format(layoutUS)
					}
					buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateStringEllipsis(p.Name, 20), sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID, false), 20), lseen, joined, fact.LevelToString(p.Level))
					count++
				}
			}
			if count <= 0 {
				buf = "No results."
			}

			//1 << 6 is ephemeral/private
			respData := &discordgo.InteractionResponseData{Content: buf, Flags: 1 << 6}
			resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
			err := s.InteractionRespond(i.Interaction, resp)
			if err != nil {
				cwlog.DoLogCW(err.Error())
			}
			return
		}
	}
	disc.EphemeralResponse(s, i, "Error:", "You must supply a search term.")

}
