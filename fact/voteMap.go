package fact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

/* See if the player's vote is valid and add it to the list */
func CheckVote(i *discordgo.InteractionCreate, arg string) {

	if strings.EqualFold(arg, "new-map") ||
		strings.EqualFold(arg, "skip-reset") {
		if !cfg.Local.Options.MembersOnly &&
			!cfg.Local.Options.RegularsOnly {
			return
		}
	}

	time.Sleep(time.Second)
	glob.VoteBoxLock.Lock()
	defer glob.VoteBoxLock.Unlock()

	if !FactorioBooted || !FactIsRunning {
		buf := "Factorio is not running."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(i, &f)
		return
	}

	/* Only if allowed */
	if !disc.CheckRegular(i) && !disc.CheckModerator(i) && !disc.CheckAdmin(i) {
		buf := "You must have the `" + strings.ToUpper(cfg.Global.Discord.Roles.Regular) + "` Discord role to use this command. See /register and the read-this-first channel for more info."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(i, &f)
		return
	}

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	/* Check if file is valid and found */
	autoSaveStr := fmt.Sprintf("%v.zip", arg)
	_, err := os.Stat(path + "/" + autoSaveStr)
	notfound := os.IsNotExist(err)

	/* Just in case people bypass Discord */
	if !strings.HasSuffix(autoSaveStr, ".zip") && strings.HasSuffix(autoSaveStr, "tmp.zip") && strings.HasSuffix(autoSaveStr, cfg.Local.Name+"_new.zip") {
		notfound = true
	}

	if !strings.EqualFold(arg, "new-map") &&
		!strings.EqualFold(arg, "skip-reset") {
		if notfound {

			buf := "That save doesn't exist."
			f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
			disc.FollowupResponse(i, &f)
			return
		}

		good, _ := CheckSave(path, autoSaveStr, false)
		if !good {
			buf := fmt.Sprintf("The save game '%v' does not appear to be valid.", autoSaveStr)
			f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
			disc.FollowupResponse(i, &f)
			return
		}
	}

	/* Cooldown */
	if !glob.VoteBox.LastMapChange.IsZero() && time.Since(glob.VoteBox.LastMapChange) < constants.MapCooldownMins*time.Minute {
		left := (constants.MapCooldownMins * time.Minute).Round(time.Second) - time.Since(glob.VoteBox.LastMapChange)
		buf := fmt.Sprintf("The map can not be changed for another %v.", left.Round(time.Second).String())
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(i, &f)
		return
	}

	/* Autosave exists, handle votes */
	var v glob.MapVoteData = glob.MapVoteData{}
	vpos := 0
	changedVote := false
	foundVote := false
	TallyMapVotes()
	for vpos, v = range glob.VoteBox.Votes {
		if strings.EqualFold(v.DiscID, i.Member.User.ID) && strings.EqualFold(v.Name, i.Member.User.Username) {
			left := (constants.VoteCooldown * time.Minute).Round(time.Second) - time.Since(v.Time)

			if v.Selection != arg && !v.Voided && v.NumChanges < constants.MaxVoteChanges {
				glob.VoteBox.Votes[vpos].Selection = arg
				glob.VoteBox.Votes[vpos].Time = time.Now()
				glob.VoteBox.Votes[vpos].Voided = false
				glob.VoteBox.Votes[vpos].Expired = false
				glob.VoteBox.Votes[vpos].NumChanges++
				glob.VoteBox.Votes[vpos].TotalVotes++
				var buf string
				buf = fmt.Sprintf("You have changed your vote to: %v", arg)

				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(i, &f)
				changedVote = true

				buf = fmt.Sprintf("%v has changed their vote to: %v", i.Member.User.Username, arg)

				CMS(cfg.Local.Channel.ChatChannel, buf)
				break
			} else if v.NumChanges >= constants.MaxVoteChanges {
				buf := "You can not change your vote anymore until it expires."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(i, &f)
				return
			}

			/* If they didn't change a already valid vote, then check cooldown */
			if left > 0 && !changedVote {
				buf := "You can not vote again yet, you must wait " + left.Round(time.Second).String() + "."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(i, &f)
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

		newVote := glob.MapVoteData{Name: i.Member.User.Username,
			DiscID: i.Member.User.ID, TotalVotes: 0, Time: time.Now(),
			Selection: arg, NumChanges: 0, Voided: false, Expired: false,
			Moderator: disc.CheckModerator(i), Supporter: disc.CheckSupporter(i), Veteran: disc.CheckVeteran(i)}

		/* Re-use old vote if we found one, or old votes will block new ones */
		if foundVote && len(glob.VoteBox.Votes) >= vpos { /* sanity check */
			if !disc.CheckModerator(i) && glob.VoteBox.Votes[vpos].TotalVotes >= constants.MaxVotesPerMap {
				buf := "You have used all of your allotted votes for this cycle."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(i, &f)
				return
			} else {
				glob.VoteBox.Votes[vpos] = newVote
				glob.VoteBox.Votes[vpos].TotalVotes++
			}
		} else if !changedVote {
			glob.VoteBox.Votes = append(glob.VoteBox.Votes, newVote)
		}
		var buf string

		buf = fmt.Sprintf("You have voted for: %v", arg)
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(i, &f)

		buf = fmt.Sprintf("%v has voted for: %v", i.Member.User.Username, arg)
		CMS(cfg.Local.Channel.ChatChannel, buf)

	}

	/* Count and show votes */
	str, count := TallyMapVotes()
	if count > 0 {
		CMS(cfg.Local.Channel.ChatChannel, str)
	}

	found := false
	var chosenMap string
	for _, t := range glob.VoteBox.Tally {
		if t.Count >= constants.VotesNeeded {
			msg := fmt.Sprintf("%v-%v: Players voted for: %v", cfg.Local.Callsign, cfg.Local.Name, t.Selection)

			CMS(cfg.Global.Discord.ReportChannel, msg)
			found = true
			chosenMap = t.Selection
			break
		}
	}
	/* Nope, not enough votes for one specific autosave */
	if !found {
		return
	}

	/* Set cooldown */
	glob.VoteBox.LastMapChange = time.Now().UTC().Round(time.Second)

	/* Count number of changes, for future use */
	glob.VoteBox.NumChanges++

	VoidAllVotes()
	WriteVotes()

	CMS(cfg.Local.Channel.ChatChannel, "VOTE MAP: "+chosenMap)
	FactorioBootedAt = time.Time{}
	DoChangeMap(chosenMap)
}

/* Don't use if already locked */
func VoidAllVotes() {
	/* Clear votes */
	glob.VoteBox.Votes = []glob.MapVoteData{}
	glob.VoteBox.Tally = []glob.VoteTallyData{}

}

func PrintVote(v glob.MapVoteData) string {
	typeStr := ""
	points := 1
	if v.Moderator {
		typeStr = typeStr + " (Moderator)"
		points = 2
	} else if v.Supporter {
		typeStr = typeStr + " (Supporter)"
		points = 2
	}

	buf := fmt.Sprintf("%v: Points: %v: %v (%v ago)%v", v.Name, points, v.Selection, time.Since(v.Time).Round(time.Second).String(), typeStr)
	return buf
}

/* EXPECTS LOCKED VOTEBOX */
func TallyMapVotes() (string, int) {
	validVotes := 0
	visVotes := 0

	glob.VoteBox.Tally = []glob.VoteTallyData{}

	buf := "VOTE-MAP: votes cast\n```"
	for vpos, v := range glob.VoteBox.Votes {

		/* Void or Cast */
		if v.Voided {
			buf = buf + PrintVote(v)
			buf = buf + " (VOID)\n"
			glob.VoteBox.Votes[vpos].NumChanges = 0
			visVotes++

			/* Expired */
		} else if (time.Since(v.Time) > (constants.VoteExpire*time.Hour) || v.Expired) && !v.Voided {
			glob.VoteBox.Votes[vpos].Expired = true
			glob.VoteBox.Votes[vpos].NumChanges = 0
			buf = buf + PrintVote(v)
			buf = buf + " (EXPIRED)\n"
			visVotes++

			/* Valid */
		} else if !v.Voided && !v.Expired {
			buf = buf + PrintVote(v)
			buf = buf + "\n"
			visVotes++
			validVotes++
			if v.Moderator || v.Supporter || v.Veteran {
				validVotes++
			}

		}
	}
	buf = buf + " ```\n"

	/* Clear if nothing generated */
	if visVotes == 0 {
		buf = ""
	}
	buf = buf + fmt.Sprintf("`Points: %v -- (need %v of same selection)`\n", validVotes, constants.VotesNeeded)

	/* Reset tally, recount */
	glob.VoteBox.Tally = []glob.VoteTallyData{}
	for _, v := range glob.VoteBox.Votes {
		skipAdd := false

		if !v.Voided && !v.Expired {
			for apos, a := range glob.VoteBox.Tally {
				if v.Selection == a.Selection {

					/* Same autosave, tally */
					if v.Moderator || v.Supporter || v.Veteran {
						/* Supporters and mods get two votes */
						glob.VoteBox.Tally[apos] = glob.VoteTallyData{Selection: a.Selection, Count: a.Count + 2}
					} else {
						glob.VoteBox.Tally[apos] = glob.VoteTallyData{Selection: a.Selection, Count: a.Count + 1}
					}
					skipAdd = true
				}
			}

			if !skipAdd {
				/* Different autosave, add to list */
				if v.Moderator || v.Supporter || v.Veteran {
					glob.VoteBox.Tally = append(glob.VoteBox.Tally, glob.VoteTallyData{Selection: v.Selection, Count: 2})
				} else {
					glob.VoteBox.Tally = append(glob.VoteBox.Tally, glob.VoteTallyData{Selection: v.Selection, Count: 1})
				}
			}
		}
	}

	buf = buf + "If you have the `" + strings.ToUpper(cfg.Global.Discord.Roles.Regular) + "` Discord role, use `/vote-map` to vote."
	return buf, validVotes
}

/* Expects locked votebox */
func WriteVotes() bool {

	finalPath := constants.VoteFile
	tempPath := constants.VoteFile + "." + cfg.Local.Callsign + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	glob.VoteBox.Version = "0.0.1"

	if err := enc.Encode(glob.VoteBox); err != nil {
		cwlog.DoLogCW("WriteVotes: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("WriteVotes: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteVotes: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename VoteFile file.")
		return false
	}

	return true
}

/* Read json file containing votes */
func ReadVotes() bool {
	_, err := os.Stat(constants.VoteFile)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadVotes: os.Stat failed")
		return true
	} else { /* Just read the config */

		file, err := os.ReadFile(constants.VoteFile)

		if file != nil && err == nil {
			temp := CreateVoteContainer()

			err := json.Unmarshal([]byte(file), &temp)
			if err != nil {
				cwlog.DoLogCW("ReadVotes: Unmarshal failure")
				cwlog.DoLogCW(err.Error())
				return false
			}

			glob.VoteBox = temp
			return true
		} else {
			cwlog.DoLogCW("ReadVotes: ReadFile failure")
			return false
		}

	}
}

func CreateVoteContainer() glob.VoteContainerData {

	temp := glob.VoteContainerData{
		Version:       "0.0.1",
		Votes:         []glob.MapVoteData{},
		LastMapChange: time.Now().UTC().Round(time.Second),
		NumChanges:    0,
	}
	return temp
}
