package fact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func CheckRewindVote(s *discordgo.Session, i *discordgo.InteractionCreate, argStr string) {

	glob.VoteBoxLock.Lock()
	defer glob.VoteBoxLock.Unlock()

	arg, err := strconv.Atoi(argStr)
	if err != nil {
		buf := "That is not a valid number."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
		return
	}

	if !FactorioBooted || !FactIsRunning {
		buf := "Factorio is not running."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
		return
	}

	/* Only if allowed */
	if !disc.CheckRegular(i.Member.Roles) && !disc.CheckModerator(i.Member.Roles, i) {
		buf := "You must have the `" + strings.ToUpper(cfg.Global.Discord.Roles.Regular) + "` Discord role to use this command. See /register and the read-this-first channel for more info."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
		return
	}

	/* Correct number of arguments (1) */

	if arg > 0 || arg < (cfg.Global.Options.AutosaveMax) {
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
			buf := "That autosave doesn't exist."
			f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
			disc.FollowupResponse(s, i, &f)
			return
		}
	} else {
		buf := "Not an acceptable autosave number."
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
		return
	}

	/* Cooldown */
	if time.Since(glob.VoteBox.LastRewindTime) < constants.RewindCooldownMinutes*time.Minute {
		left := (constants.RewindCooldownMinutes * time.Minute).Round(time.Second) - time.Since(glob.VoteBox.LastRewindTime)
		buf := fmt.Sprintf("The map can not be rewound for another %v.", left.Round(time.Second).String())
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
		return
	}

	/* Autosave exists, handle votes */
	var v glob.RewindVoteData = glob.RewindVoteData{}
	vpos := 0
	changedVote := false
	foundVote := false
	TallyRewindVotes()
	for vpos, v = range glob.VoteBox.Votes {
		if strings.EqualFold(v.DiscID, i.Member.User.ID) && strings.EqualFold(v.Name, i.Member.User.Username) {
			left := (constants.VoteLifetime * time.Minute).Round(time.Second) - time.Since(v.Time)

			if v.AutosaveNum != int(arg) && !v.Voided && v.NumChanges < constants.MaxRewindChanges {
				glob.VoteBox.Votes[vpos].AutosaveNum = int(arg)
				glob.VoteBox.Votes[vpos].Time = time.Now()
				glob.VoteBox.Votes[vpos].Voided = false
				glob.VoteBox.Votes[vpos].Expired = false
				glob.VoteBox.Votes[vpos].NumChanges++
				glob.VoteBox.Votes[vpos].TotalVotes++
				buf := fmt.Sprintf("You have changed your vote to autosave #%v", arg)
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(s, i, &f)
				TallyRewindVotes()
				changedVote = true
				break
			} else if v.NumChanges >= constants.MaxRewindChanges {
				buf := "You can not change your vote anymore until it expires."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(s, i, &f)
				return
			}

			/* If they didn't change a already valid vote, then check cooldown */
			if left > 0 && !changedVote {
				buf := "You can not vote again yet, you must wait " + left.Round(time.Second).String() + "."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(s, i, &f)
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

		newVote := glob.RewindVoteData{Name: i.Member.User.Username, DiscID: i.Member.User.ID, TotalVotes: 0, Time: time.Now(), AutosaveNum: int(arg), NumChanges: 0, Voided: false, Expired: false}

		/* Re-use old vote if we found one, or old votes will block new ones */
		if foundVote && len(glob.VoteBox.Votes) >= vpos { /* sanity check */
			if glob.VoteBox.Votes[vpos].TotalVotes >= constants.MaxVotesPerMap {
				buf := "You have used all of your allotted votes for this map."
				f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
				disc.FollowupResponse(s, i, &f)
				return
			} else {
				glob.VoteBox.Votes[vpos] = newVote
				glob.VoteBox.Votes[vpos].TotalVotes++
			}
		} else if !changedVote {
			glob.VoteBox.Votes = append(glob.VoteBox.Votes, newVote)
		}
		buf := fmt.Sprintf("You have voted to rewind the map to autosave #%v", arg)
		f := discordgo.WebhookParams{Content: buf, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)
	}

	/* Mark dirty, so vote is saved after we are done here */
	glob.VoteBox.Dirty = true
	if buf, c := TallyRewindVotes(); c < constants.VotesNeededRewind {
		CMS(cfg.Local.Channel.ChatChannel, buf)
		return
	} else {
		/* Enough votes to check, lets tally them and see if there is a winner */
		TallyRewindVotes() /* Updates votes */

		found := false
		asnum := 0
		for _, t := range glob.VoteBox.Tally {
			if t.Count >= constants.VotesNeededRewind {
				msg := fmt.Sprintf("%v-%v: Players voted to rewind the map to autosave #%v", cfg.Local.Callsign, cfg.Local.Name, t.Autosave)
				CMS(cfg.Global.Discord.ReportChannel, msg)
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
		CMS(cfg.Local.Channel.ChatChannel, "VOTE REWIND: Rewinding the map to autosave #"+strconv.Itoa(asnum))
		DoRewindMap(s, strconv.Itoa(asnum))
		return
	}

}

func ResetTotalVotes() {
	for vpos := range glob.VoteBox.Votes {
		glob.VoteBox.Votes[vpos].TotalVotes = 0
	}
	TallyRewindVotes()
	WriteRewindVotes()
}

/* Don't use if already locked */
func VoidAllVotes() {
	for vpos := range glob.VoteBox.Votes {
		glob.VoteBox.Votes[vpos].Voided = true
	}
	TallyRewindVotes()
	WriteRewindVotes()
}

func PrintVote(v glob.RewindVoteData) string {
	buf := fmt.Sprintf("%v: autosave #%v (%v ago)", v.Name, v.AutosaveNum, time.Since(v.Time).Round(time.Second).String())
	return buf
}

/* EXPECTS LOCKED VOTEBOX */
func TallyRewindVotes() (string, int) {
	validVotes := 0
	visVotes := 0
	totalVotes := 0

	buf := "Votes:\n```"
	for vpos, v := range glob.VoteBox.Votes {

		/* Void or Cast */
		if v.Voided {
			buf = buf + PrintVote(v)
			buf = buf + " (void/cast)\n"
			glob.VoteBox.Votes[vpos].NumChanges = 0
			visVotes++
			totalVotes++

			/* Expired */
		} else if (time.Since(v.Time) > (constants.VoteLifetime*time.Minute) || v.Expired) && !v.Voided {
			glob.VoteBox.Votes[vpos].Expired = true
			glob.VoteBox.Votes[vpos].NumChanges = 0

			totalVotes++

			/* Valid */
		} else if !v.Voided && !v.Expired {
			buf = buf + PrintVote(v)
			buf = buf + " (VALID)\n"
			visVotes++
			validVotes++
			totalVotes++
		}
	}
	buf = buf + " ```\n"

	/* Clear if nothing generated */
	if visVotes == 0 {
		buf = ""
	}
	buf = buf + fmt.Sprintf("`Total valid votes: %v -- (need %v for the same autosave)`\n", validVotes, constants.VotesNeededRewind)

	/* Reset tally, recount */
	glob.VoteBox.Tally = []glob.VoteTallyData{}
	for _, v := range glob.VoteBox.Votes {
		for apos, a := range glob.VoteBox.Tally {
			if v.AutosaveNum == a.Autosave {
				/* Same autosave, tally */
				glob.VoteBox.Tally[apos] = glob.VoteTallyData{Autosave: a.Autosave, Count: a.Count + 1}
				continue
			}
		}
		/* Different autosave, add to list */
		glob.VoteBox.Tally = append(glob.VoteBox.Tally, glob.VoteTallyData{Autosave: v.AutosaveNum, Count: 1})
	}

	buf = buf + "Syntax: `$vote-rewind <autosave number>`"
	return buf, validVotes
}

/* Expects locked votebox */
func WriteRewindVotes() bool {
	finalPath := constants.VoteRewindFile
	tempPath := constants.VoteRewindFile + "." + cfg.Local.Callsign + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	glob.VoteBox.Version = "0.0.1"

	if err := enc.Encode(glob.VoteBox); err != nil {
		cwlog.DoLogCW("WriteRewindVotes: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("WriteRewindVotes: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteRewindVotes: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename VoteRewindFile file.")
		return false
	}

	return true
}

func ReadRewindVotes() bool {
	_, err := os.Stat(constants.VoteRewindFile)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadRewindVotes: os.Stat failed")
		return true
	} else { /* Just read the config */

		file, err := ioutil.ReadFile(constants.VoteRewindFile)

		if file != nil && err == nil {
			temp := CreateVoteContainer()

			err := json.Unmarshal([]byte(file), &temp)
			if err != nil {
				cwlog.DoLogCW("ReadRewindVotes: Unmarshal failure")
				cwlog.DoLogCW(err.Error())
				return false
			}

			glob.VoteBox = temp
			return true
		} else {
			cwlog.DoLogCW("ReadRewindVotes: ReadFile failure")
			return false
		}

	}
}

func CreateVoteContainer() glob.RewindVoteContainerData {

	temp := glob.RewindVoteContainerData{
		Version:        "0.0.1",
		Votes:          []glob.RewindVoteData{},
		LastRewindTime: time.Now().Round(time.Second),
		NumRewind:      0,
	}
	return temp
}
