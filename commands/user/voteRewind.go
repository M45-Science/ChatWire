package user

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

/* Allow regulars to vote to rewind the map */
func VoteRewind(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")

	glob.VoteBoxLock.Lock()
	defer glob.VoteBoxLock.Unlock()

	argnum := len(args)

	/* Mod commands */
	if disc.CheckModerator(i.Member.Roles) {
		if argnum > 0 {
			if strings.EqualFold(args[0], "erase") {
				/* Clear votes */
				glob.VoteBox.Votes = []glob.RewindVoteData{}

				fact.CMS(cfg.Local.Channel.ChatChannel, "All votes cleared.")
				fact.TallyRewindVotes()
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "void") {
				/* Void votes */
				for vpos := range glob.VoteBox.Votes {
					glob.VoteBox.Votes[vpos].Voided = true
				}
				fact.CMS(cfg.Local.Channel.ChatChannel, "All votes voided.")
				fact.TallyRewindVotes()
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "show") {
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
			} else if strings.EqualFold(args[0], "tally") {
				/* Retally votes */
				fact.TallyRewindVotes()
				fact.CMS(cfg.Local.Channel.ChatChannel, "Votes re-tallied (debug).")
				return
			} else if strings.EqualFold(args[0], "save") {
				/* Force save */
				fact.CMS(cfg.Local.Channel.ChatChannel, "Votes database saved.")
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "reset-cooldown") {
				/* Reset cooldown */
				glob.VoteBox.LastRewindTime = time.Now()
				fact.CMS(cfg.Local.Channel.ChatChannel, "Cooldown reset.")
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "no-cooldown") {
				/* Reset cooldown */
				glob.VoteBox.LastRewindTime = time.Now().Add(time.Duration((-constants.RewindCooldownMinutes) * time.Minute))
				fact.CMS(cfg.Local.Channel.ChatChannel, "Cooldown removed.")
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "cooldown") {
				/* 60m cooldown */
				glob.VoteBox.LastRewindTime = time.Now().Add(time.Duration((60 - constants.RewindCooldownMinutes) * time.Minute))
				fact.CMS(cfg.Local.Channel.ChatChannel, "Cooldown set to 60m")
				fact.WriteRewindVotes()
				return
			} else if strings.EqualFold(args[0], "help") {
				/* Show help */
				fact.CMS(cfg.Local.Channel.ChatChannel, "`vote-rewind erase` to erase all votes\n`vote-rewind void` to void all votes (no re-voting for 15m)\n`vote-rewind show` to display whole database\n`vote-rewind tally` to re-tally votes (debug)\n`vote-rewind save` to force-save votes\n`vote-rewind reset-cooldown` to disallow rewinding for a few minutes\n`vote-rewind cooldown` to disallow rewinding for 1 hour\n`vote-rewind no-cooldown` to allow rewinding again immediately.")
				return
			}
		}
	}

	if !fact.IsFactorioBooted() || !fact.IsFactRunning() || !glob.ServerRunning {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio isn't running!")
		return
	}

	/* Only if allowed */
	if !disc.CheckRegular(i.Message.Member.Roles) && !disc.CheckModerator(i.Message.Member.Roles) {
		fact.CMS(cfg.Local.Channel.ChatChannel, "You must have the `"+strings.ToUpper(cfg.Global.Discord.Roles.Regular)+"` Discord role to use this command. See $help register for more info.")
		return
	}

	var err error
	if argnum > 0 {
		_, err = strconv.Atoi(args[0])
	}
	if argnum == 0 || err != nil {
		fact.ShowRewindList(s)
		buf, _ := fact.TallyRewindVotes()
		if buf != "" {
			fact.CMS(cfg.Local.Channel.ChatChannel, buf)
		}
		return
	}
	/* Correct number of arguments (1) */
	if argnum == 1 {

		/* Make sure the autosave exists. */
		arg := args[0]
		arg = strings.TrimSpace(arg)
		arg = strings.TrimPrefix(arg, "#")
		num, err := strconv.Atoi(arg)

		if err != nil {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Not a valid autosave number.")
			return
		}
		if num > 0 || num < 9999 {
			path := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.FactorioPrefix + cfg.Local.Callsign + "/" + cfg.Global.Paths.Folders.Saves
			/* Check if file is valid and found */
			autoSaveStr := fmt.Sprintf("_autosave%v.zip", num)
			_, err := os.Stat(path + "/" + autoSaveStr)
			notfound := os.IsNotExist(err)

			if notfound {
				fact.CMS(cfg.Local.Channel.ChatChannel, "That autosave number does not exist.")
				return
			}
		} else {
			fact.CMS(cfg.Local.Channel.ChatChannel, "Not a valid autosave number.")
			return
		}

		/* Cooldown */
		if time.Since(glob.VoteBox.LastRewindTime) < constants.RewindCooldownMinutes*time.Minute {
			left := (constants.RewindCooldownMinutes * time.Minute).Round(time.Second) - time.Since(glob.VoteBox.LastRewindTime)
			fact.CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("The map can not be rewound for another %v.", left.Round(time.Second).String()))
			return
		}

		/* Autosave exists, handle votes */
		var v glob.RewindVoteData = glob.RewindVoteData{}
		vpos := 0
		changedVote := false
		foundVote := false
		fact.TallyRewindVotes()
		for vpos, v = range glob.VoteBox.Votes {
			if v.DiscID == i.Message.Author.ID {
				left := (constants.VoteLifetime * time.Minute).Round(time.Second) - time.Since(v.Time)

				if v.AutosaveNum != num && !v.Voided && v.NumChanges < constants.MaxRewindChanges {
					glob.VoteBox.Votes[vpos].AutosaveNum = num
					glob.VoteBox.Votes[vpos].Time = time.Now()
					glob.VoteBox.Votes[vpos].Voided = false
					glob.VoteBox.Votes[vpos].Expired = false
					glob.VoteBox.Votes[vpos].NumChanges++
					glob.VoteBox.Votes[vpos].TotalVotes++
					fact.CMS(cfg.Local.Channel.ChatChannel, "You have changed your vote to autosave #"+strconv.Itoa(num))
					fact.TallyRewindVotes()
					changedVote = true
					break
				} else if v.NumChanges >= constants.MaxRewindChanges {
					fact.CMS(cfg.Local.Channel.ChatChannel, "You can't change your vote again.")
					return
				}

				/* If they didn't change a already valid vote, then check cooldown */
				if left > 0 && !changedVote {
					fact.CMS(cfg.Local.Channel.ChatChannel, "You can not vote again yet, you must wait "+left.Round(time.Second).String()+".")
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

			newVote := glob.RewindVoteData{Name: i.Message.Author.Username, DiscID: i.Message.Author.ID, TotalVotes: 0, Time: time.Now(), AutosaveNum: num, NumChanges: 0, Voided: false, Expired: false}

			/* Re-use old vote if we found one, or old votes will block new ones */
			if foundVote && len(glob.VoteBox.Votes) >= vpos { /* sanity check */
				if glob.VoteBox.Votes[vpos].TotalVotes >= constants.MaxVotesPerMap {
					fact.CMS(cfg.Local.Channel.ChatChannel, "You are over the maximum number of votes per map.")
					return
				} else {
					glob.VoteBox.Votes[vpos] = newVote
					glob.VoteBox.Votes[vpos].TotalVotes++
				}
			} else if !changedVote {
				glob.VoteBox.Votes = append(glob.VoteBox.Votes, newVote)
			}
			fact.CMS(cfg.Local.Channel.ChatChannel, "You have voted to rewind the map to autosave #"+args[0])
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
					//fact.CMS(cfg.Local.Channel.ChatChannel, "Players voted to rewind map to autosave #"+strconv.Itoa(as.Autosave))
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
			fact.CMS(cfg.Local.Channel.ChatChannel, "Rewinding the map to autosave #"+strconv.Itoa(asnum))
			fact.DoRewindMap(s, strconv.Itoa(asnum))
			return
		}

	} else {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Invalid arguments.")
		return
	}

}
