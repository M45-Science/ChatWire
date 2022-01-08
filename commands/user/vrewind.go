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

//Allow regulars to vote to rewind the map
func VoteRewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	glob.VoteBoxLock.Lock()
	defer glob.VoteBoxLock.Unlock()

	argnum := len(args)

	if !fact.IsFactorioBooted() || !fact.IsFactRunning() || !glob.ServerRunning {
		fact.CMS(m.ChannelID, "Factorio isn't running!")
		return
	}

	//Only if allowed
	if !disc.CheckRegular(m) && !disc.CheckModerator(m) {
		fact.CMS(m.ChannelID, "You must have the `"+strings.ToUpper(cfg.Global.RoleData.RegularRoleName)+"` Discord role to use this command.")
		return
	}

	var err error
	if argnum > 0 {
		_, err = strconv.Atoi(args[0])
	}
	if argnum == 0 || err != nil {
		fact.ShowRewindList(s, m)
		buf, _ := fact.TallyRewindVotes()
		if buf != "" {
			fact.CMS(m.ChannelID, buf)
		}
		return
	}

	//Correct number of arguments (1)
	if argnum == 1 {

		//Make sure the autosave exists.
		arg := args[0]
		arg = strings.TrimSpace(arg)
		arg = strings.TrimPrefix(arg, "#")
		num, err := strconv.Atoi(arg)

		if err != nil {
			fact.CMS(m.ChannelID, "Not a valid autosave number.")
			return
		}
		if num > 0 || num < 9999 {
			path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath
			//Check if file is valid and found
			autoSaveStr := fmt.Sprintf("_autosave%v.zip", num)
			_, err := os.Stat(path + "/" + autoSaveStr)
			notfound := os.IsNotExist(err)

			if notfound {
				fact.CMS(m.ChannelID, "That autosave number does not exist.")
				return
			}
		} else {
			fact.CMS(m.ChannelID, "Not a valid autosave number.")
			return
		}

		//Cooldown
		if time.Since(glob.VoteBox.LastRewindTime) < constants.RewindCooldownMinutes*time.Minute {
			fact.CMS(m.ChannelID, "The map was rewound "+time.Since(glob.VoteBox.LastRewindTime).Round(time.Second).String()+" ago,")
			left := (constants.RewindCooldownMinutes * time.Minute).Round(time.Second) - time.Since(glob.VoteBox.LastRewindTime)
			fact.CMS(m.ChannelID, fmt.Sprintf("so it can not be rewound again for another %v", left.Round(time.Second).String()))
			return
		}

		//Autosave exists, handle votes
		var v glob.RewindVoteData = glob.RewindVoteData{}
		vpos := 0
		changedVote := false
		foundVote := false
		fact.TallyRewindVotes()
		for vpos, v = range glob.VoteBox.Votes {
			if v.DiscID == m.Author.ID {
				left := (constants.VoteLifetime * time.Minute).Round(time.Second) - time.Since(v.Time)

				if v.AutosaveNum != num && !v.Voided && v.NumChanges < constants.MaxRewindChanges {
					glob.VoteBox.Votes[vpos].AutosaveNum = num
					glob.VoteBox.Votes[vpos].Time = time.Now()
					glob.VoteBox.Votes[vpos].Voided = false
					glob.VoteBox.Votes[vpos].Expired = false
					glob.VoteBox.Votes[vpos].NumChanges++
					glob.VoteBox.Votes[vpos].TotalVotes++
					fact.CMS(m.ChannelID, "You have changed your vote to autosave #"+strconv.Itoa(num))
					fact.TallyRewindVotes()
					changedVote = true
					break
				} else if v.NumChanges >= constants.MaxRewindChanges {
					fact.CMS(m.ChannelID, "You can't change your vote again.")
					return
				}

				//If they didn't change a already valid vote, then check cooldown
				if left > 0 && !changedVote {
					fact.CMS(m.ChannelID, "You can not vote again yet, you must wait "+left.Round(time.Second).String()+".")
					return
				}

				/* Everything is good, we can cast a new vote,
				exit so we have position of existing vote */
				foundVote = true
				break
			}
		}

		//Create new vote, if we didn't already change it above
		if !changedVote {

			newVote := glob.RewindVoteData{Name: m.Author.Username, DiscID: m.Author.ID, TotalVotes: 0, Time: time.Now(), AutosaveNum: num, NumChanges: 0, Voided: false, Expired: false}

			//Re-use old vote if we found one, or old votes will block new ones
			if foundVote && len(glob.VoteBox.Votes) >= vpos { //sanity check
				glob.VoteBox.Votes[vpos] = newVote
				glob.VoteBox.Votes[vpos].TotalVotes++
			} else if !changedVote {
				glob.VoteBox.Votes = append(glob.VoteBox.Votes, newVote)
			}
			fact.CMS(m.ChannelID, "You have voted to rewind the map to autosave #"+args[0])
		}

		/* Mark dirty, so vote is saved after we are done here */
		glob.VoteBox.Dirty = true
		if buf, c := fact.TallyRewindVotes(); c < constants.VotesNeededRewind {
			fact.CMS(m.ChannelID, buf)
			return
		} else {
			//Enough votes to check, lets tally them and see if there is a winner
			fact.TallyRewindVotes() //Updates votes

			found := false
			asnum := 0
			for _, t := range glob.VoteBox.Tally {
				if t.Count >= constants.VotesNeededRewind {
					//fact.CMS(m.ChannelID, "Players voted to rewind map to autosave #"+strconv.Itoa(as.Autosave))
					found = true
					asnum = t.Autosave
				}
			}
			//Nope, not enough votes for one specific autosave
			if !found {
				return
			}

			//Set cooldown
			glob.VoteBox.LastRewindTime = time.Now().Round(time.Second)

			//Count number of rewinds, for future use
			glob.VoteBox.NumRewind++

			//Mark all votes as voided
			for vpos, _ = range glob.VoteBox.Votes {
				glob.VoteBox.Votes[vpos].Voided = true
			}
			fact.CMS(m.ChannelID, "Rewinding the map to autosave #"+strconv.Itoa(asnum))
			fact.DoRewindMap(s, m, strconv.Itoa(asnum))
			return
		}

	} else {
		fact.CMS(m.ChannelID, "Invalid arguments.")
		return
	}

}
