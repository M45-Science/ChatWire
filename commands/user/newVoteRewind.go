package user

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Allow regulars to vote to rewind the map */
func VoteRewind(s *discordgo.Session, i *discordgo.InteractionCreate) {

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
					glob.VoteBox.Votes = []glob.RewindVoteData{}

					disc.EphemeralResponse(s, i, "Status:", "All votes cleared.")
					fact.TallyRewindVotes()
					fact.WriteRewindVotes()
					return
				} else if strings.EqualFold(arg, "void") {
					/* Void votes */
					for vpos := range glob.VoteBox.Votes {
						glob.VoteBox.Votes[vpos].Voided = true
					}
					disc.EphemeralResponse(s, i, "Status:", "All votes voided.")
					fact.TallyRewindVotes()
					fact.WriteRewindVotes()
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
					fact.TallyRewindVotes()
					disc.EphemeralResponse(s, i, "Status:", "All votes re-tallied (debug).")
					return
				} else if strings.EqualFold(arg, "save") {
					/* Force save */
					disc.EphemeralResponse(s, i, "Status:", "votebox force-saved.")
					fact.WriteRewindVotes()
					return
				} else if strings.EqualFold(arg, "reset-cooldown") {
					/* Reset cooldown */
					glob.VoteBox.LastRewindTime = time.Now()
					disc.EphemeralResponse(s, i, "Status:", "Rewind cooldown reset.")
					fact.WriteRewindVotes()
					return
				} else if strings.EqualFold(arg, "no-cooldown") {
					/* Reset cooldown */
					glob.VoteBox.LastRewindTime = time.Now().Add(time.Duration((-constants.RewindCooldownMinutes) * time.Minute))
					disc.EphemeralResponse(s, i, "Status:", "Cooldown killed.")
					fact.WriteRewindVotes()
					return
				} else if strings.EqualFold(arg, "cooldown") {
					/* 60m cooldown */
					glob.VoteBox.LastRewindTime = time.Now().Add(time.Duration((60 - constants.RewindCooldownMinutes) * time.Minute))
					disc.EphemeralResponse(s, i, "Status:", "Cooldown set to 60m.")
					fact.WriteRewindVotes()
					return
				}
			}
		}
	}

	if !fact.FactorioBooted || !fact.FactIsRunning {
		disc.EphemeralResponse(s, i, "Status:", "Factorio is not running.")
		return
	}

	/* Only if allowed */
	if !disc.CheckRegular(i.Member.Roles) && !disc.CheckModerator(i.Member.Roles, i) {
		buf := "You must have the `" + strings.ToUpper(cfg.Global.Discord.Roles.Regular) + "` Discord role to use this command. See /register and the read-this-first channel for more info."
		disc.EphemeralResponse(s, i, "Notice:", buf)
		return
	}

	/* Correct number of arguments (1) */
	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionInteger {
			arg := o.IntValue()

			if arg > 0 || arg < int64(cfg.Global.Options.AutosaveMax) {
				path := cfg.Global.Paths.Folders.ServersRoot +
					cfg.Global.Paths.ChatWirePrefix +
					cfg.Local.Callsign + "/" +
					cfg.Global.Paths.Folders.FactorioDir + "/" +
					cfg.Global.Paths.Folders.Saves
				/* Check if file is valid and found */
				autoSaveStr := fmt.Sprintf("_autosave%v.zip", arg)
				_, err := os.Stat(path + "/" + autoSaveStr)
				notfound := os.IsNotExist(err)

				if notfound {
					disc.EphemeralResponse(s, i, "Error:", "That autosave doesn't exist.")
					return
				}
			} else {
				disc.EphemeralResponse(s, i, "Error:", "Not an acceptable autosave number.")
				return
			}

			/* Cooldown */
			if time.Since(glob.VoteBox.LastRewindTime) < constants.RewindCooldownMinutes*time.Minute {
				left := (constants.RewindCooldownMinutes * time.Minute).Round(time.Second) - time.Since(glob.VoteBox.LastRewindTime)
				buf := fmt.Sprintf("The map can not be rewound for another %v.", left.Round(time.Second).String())
				disc.EphemeralResponse(s, i, "Notice:", buf)
				return
			}

			/* Autosave exists, handle votes */
			var v glob.RewindVoteData = glob.RewindVoteData{}
			vpos := 0
			changedVote := false
			foundVote := false
			fact.TallyRewindVotes()
			for vpos, v = range glob.VoteBox.Votes {
				if strings.EqualFold(v.DiscID, i.Message.Author.ID) {
					left := (constants.VoteLifetime * time.Minute).Round(time.Second) - time.Since(v.Time)

					if v.AutosaveNum != int(arg) && !v.Voided && v.NumChanges < constants.MaxRewindChanges {
						glob.VoteBox.Votes[vpos].AutosaveNum = int(arg)
						glob.VoteBox.Votes[vpos].Time = time.Now()
						glob.VoteBox.Votes[vpos].Voided = false
						glob.VoteBox.Votes[vpos].Expired = false
						glob.VoteBox.Votes[vpos].NumChanges++
						glob.VoteBox.Votes[vpos].TotalVotes++
						buf := fmt.Sprintf("You have changed your vote to autosave #%v", arg)
						disc.EphemeralResponse(s, i, "Error:", buf)
						fact.TallyRewindVotes()
						changedVote = true
						break
					} else if v.NumChanges >= constants.MaxRewindChanges {
						disc.EphemeralResponse(s, i, "Error:", "You can not change your vote anymore until it expires.")
						return
					}

					/* If they didn't change a already valid vote, then check cooldown */
					if left > 0 && !changedVote {
						buf := "You can not vote again yet, you must wait " + left.Round(time.Second).String() + "."
						disc.EphemeralResponse(s, i, "Error:", buf)
						return
					}

					/* Everything is good, we can cast a new vote,
					exit so we have position of existing vote */
					foundVote = true
					break
				}
			}

			/* Create new vote, if we didn't already change it above */
			if !changedVote {

				newVote := glob.RewindVoteData{Name: i.Message.Author.Username, DiscID: i.Message.Author.ID, TotalVotes: 0, Time: time.Now(), AutosaveNum: int(arg), NumChanges: 0, Voided: false, Expired: false}

				/* Re-use old vote if we found one, or old votes will block new ones */
				if foundVote && len(glob.VoteBox.Votes) >= vpos { /* sanity check */
					if glob.VoteBox.Votes[vpos].TotalVotes >= constants.MaxVotesPerMap {
						disc.EphemeralResponse(s, i, "Error:", "You have used all of your allotted votes for this map.")
						return
					} else {
						glob.VoteBox.Votes[vpos] = newVote
						glob.VoteBox.Votes[vpos].TotalVotes++
					}
				} else if !changedVote {
					glob.VoteBox.Votes = append(glob.VoteBox.Votes, newVote)
				}
				buf := fmt.Sprintf("You have voted to rewind the map to autosave #%v", arg)
				disc.EphemeralResponse(s, i, "Notice:", buf)
			}

			/* Mark dirty, so vote is saved after we are done here */
			glob.VoteBox.Dirty = true
			if buf, c := fact.TallyRewindVotes(); c < constants.VotesNeededRewind {
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
				return
			} else {
				/* Enough votes to check, lets tally them and see if there is a winner */
				fact.TallyRewindVotes() /* Updates votes */

				found := false
				asnum := 0
				for _, t := range glob.VoteBox.Tally {
					if t.Count >= constants.VotesNeededRewind {
						msg := fmt.Sprintf("%v-%v: Players voted to rewind the map to autosave #%v", cfg.Local.Callsign, cfg.Local.Name, t.Autosave)
						fact.CMS(cfg.Global.Discord.ReportChannel, msg)
						found = true
						asnum = t.Autosave
					}
				}
				/* Nope, not enough votes for one specific autosave */
				if !found {
					return
				}

				/* Set cooldown */
				glob.VoteBox.LastRewindTime = time.Now().Round(time.Second)

				/* Count number of rewinds, for future use */
				glob.VoteBox.NumRewind++

				/* Mark all votes as voided */
				for vpos = range glob.VoteBox.Votes {
					glob.VoteBox.Votes[vpos].Voided = true
				}
				fact.CMS(cfg.Local.Channel.ChatChannel, "VOTE REWIND: Rewinding the map to autosave #"+strconv.Itoa(asnum))
				fact.DoRewindMap(s, strconv.Itoa(asnum))
				return
			}
		}
	}
}
