package user

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
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
				if strings.EqualFold(arg, "erase") {
					/* Clear votes */
					glob.VoteBox.Votes = []glob.MapVoteData{}

					disc.EphemeralResponse(s, i, "Status:", "All votes cleared.")
					fact.TallyMapVotes()
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "void") {
					/* Void votes */
					for vpos := range glob.VoteBox.Votes {
						glob.VoteBox.Votes[vpos].Voided = true
					}
					disc.EphemeralResponse(s, i, "Status:", "All votes voided.")
					fact.TallyMapVotes()
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "show") {
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
				} else if strings.EqualFold(arg, "tally") {
					/* Retally votes */
					fact.TallyMapVotes()
					disc.EphemeralResponse(s, i, "Status:", "All votes re-tallied (debug).")
					return
				} else if strings.EqualFold(arg, "save") {
					/* Force save */
					disc.EphemeralResponse(s, i, "Status:", "votebox force-saved.")
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "reset-cooldown") {
					/* Reset cooldown */
					glob.VoteBox.LastMapChange = time.Now()
					disc.EphemeralResponse(s, i, "Status:", "Cooldown reset.")
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "no-cooldown") {
					/* Reset cooldown */
					glob.VoteBox.LastMapChange = time.Now().Add(time.Duration((-constants.MapCooldownMins) * time.Minute))
					disc.EphemeralResponse(s, i, "Status:", "Cooldown killed.")
					fact.WriteVotes()
					return
				} else if strings.EqualFold(arg, "cooldown") {
					/* 60m cooldown */
					glob.VoteBox.LastMapChange = time.Now().Add(time.Duration((60 - constants.MapCooldownMins) * time.Minute))
					disc.EphemeralResponse(s, i, "Status:", "Cooldown set to 60m.")
					fact.WriteVotes()
					return
				}
			}
		}
	}
	fact.ShowMapList(s, i, true)

}
