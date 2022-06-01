package user

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Allow regulars to vote to change the map*/
func VoteMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	glob.VoteBoxLock.Lock()
	defer glob.VoteBoxLock.Unlock()

	a := i.ApplicationCommandData()

	/* Mod commands */
	if disc.CheckModerator(i.Member.Roles, i) {
		for _, o := range a.Options {
			if o.Type == discordgo.ApplicationCommandOptionString {
				arg := o.StringValue()
				if strings.EqualFold(arg, "erase-all") {
					/* Clear votes */
					glob.VoteBox.Votes = []glob.MapVoteData{}

					disc.EphemeralResponse(s, i, "Status:", "All votes cleared.")
					fact.TallyMapVotes()
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "void-all") {
					/* Void votes */
					for vpos := range glob.VoteBox.Votes {
						glob.VoteBox.Votes[vpos].Voided = true
					}
					disc.EphemeralResponse(s, i, "Status:", "All votes voided.")
					fact.TallyMapVotes()
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "show-all") {
					/* Show votes */
					buf := "Votes: ``` \n"
					for _, tvote := range glob.VoteBox.Votes {
						tags := ""
						if tvote.Voided {
							tags = " (void/cast)"
						}
						if tvote.Expired {
							tags = tags + " (expired)"
						}
						buf = buf + fact.PrintVote(tvote)
						buf = buf + tags + "\n"
					}
					buf = buf + " \n```"
					fact.CMS(cfg.Local.Channel.ChatChannel, buf)
					return
				}
			}
		}
	}
	fact.ShowMapList(s, i, true)

}
