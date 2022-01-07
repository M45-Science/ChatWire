package user

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type rewindVoteData struct {
	Name        string
	DiscID      string
	AutosaveNum int

	Time    time.Time
	Voided  bool
	Expired bool
}

var votes []rewindVoteData
var autoSaveList []asData
var LastRewindTime time.Time
var numRewind int = 0

//Allow regulars to vote to rewind the map
func VoteRewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
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
		buf, _ := getVotes()
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
		if time.Since(LastRewindTime) < constants.RewindCooldownMinutes*time.Minute {
			fact.CMS(m.ChannelID, "The map was rewound "+time.Since(LastRewindTime).Round(time.Second).String()+" ago,")
			left := (constants.RewindCooldownMinutes * time.Minute).Round(time.Second) - time.Since(LastRewindTime)
			fact.CMS(m.ChannelID, fmt.Sprintf("so it can not be rewound again for another %v", left.Round(time.Second).String()))
			return
		}

		//Autosave exists, handle votes
		var v rewindVoteData = rewindVoteData{}
		vpos := 0
		usedExistingVote := false
		for vpos, v = range votes {
			if v.DiscID == m.Author.ID || v.Name == m.Author.Username {
				left := (constants.VoteLifetime * time.Minute).Round(time.Second) - time.Since(v.Time)

				if v.AutosaveNum != num && !v.Voided {
					votes[vpos].AutosaveNum = num
					votes[vpos].Time = time.Now()
					votes[vpos].Voided = false
					votes[vpos].Expired = false
					fact.CMS(m.ChannelID, "You have changed your vote to autosave #"+strconv.Itoa(num))
					getVotes()
					usedExistingVote = true
				}

				if left > 0 {
					fact.CMS(m.ChannelID, "You can not vote again yet, you must wait another "+left.Round(time.Second).String())
				}

				return
			}
		}

		//Create new vote
		if !usedExistingVote {
			newVote := rewindVoteData{Name: m.Author.Username, DiscID: m.Author.ID, Time: time.Now(), AutosaveNum: num}
			votes = append(votes, newVote)
			fact.CMS(m.ChannelID, "You have voted to rewind the map to autosave #"+args[0])
		}

		WriteRewindVotes()
		if _, c := getVotes(); c < constants.VotesNeededRewind {
			//Not enough votes yet
			buf, _ := getVotes()
			fact.CMS(m.ChannelID, buf)
			return
		} else {
			//Enough votes, check them.
			getVotes() //Updates autoSaveList
			found := false
			asnum := 0
			for _, as := range autoSaveList {
				if as.Count >= constants.VotesNeededRewind {
					fact.CMS(m.ChannelID, "Players voted to rewind map to autosave #"+strconv.Itoa(as.Autosave))
					found = true
					asnum = as.Autosave
				}
			}
			//Nope, exit
			if !found {
				return
			}

			//Set cooldown
			LastRewindTime = time.Now().Round(time.Second)

			//Count number of rewinds, for longer cooldowns
			numRewind++

			//Mark all votes as voided
			for _, v := range votes {
				v.Voided = true
			}
			fact.CMS(m.ChannelID, "Rewinding the map to autosave #"+strconv.Itoa(asnum))
			WriteRewindVotes()
			fact.DoRewindMap(s, m, args[0])
			return
		}

	}

}

type asData struct {
	Autosave int
	Count    int
}

func getVotes() (string, int) {
	count := 0
	buf := "Votes:\n```"
	for _, v := range votes {
		if !v.Voided {
			if time.Since(v.Time) > (constants.VoteLifetime * time.Minute) {
				//Expired vote, void it
				v.Voided = true
				v.Expired = true
				buf = buf + v.Name + " (Expired)\n"
			} else {
				buf = buf + fmt.Sprintf("%v: autosave #%v (%v ago)\n", v.Name, v.AutosaveNum, time.Since(v.Time).Round(time.Second).String())
				count++
			}
		} else {
			buf = buf + v.Name + " (Voided)\n"
		}
		if time.Since(v.Time) > (constants.VoteLifetime * time.Minute) {
			//Expired vote, void it
			v.Voided = true
			v.Expired = true
			buf = buf + v.Name + " (Expired)\n"
		}
	}
	buf = buf + fmt.Sprintf("```\n`Valid votes: %v -- (need %v total)`\n", count, constants.VotesNeededRewind)

	autoSaveList = []asData{}
	for _, v := range votes {
		for apos, a := range autoSaveList {
			if v.AutosaveNum == a.Autosave {
				autoSaveList[apos] = asData{Autosave: a.Autosave, Count: a.Count + 1}
				continue
			}
		}
		autoSaveList = append(autoSaveList, asData{Autosave: v.AutosaveNum, Count: 1})
	}

	//if len(autoSaveList) > 1 {
	//	buf = buf + "Number of votes for each autosave:\n```"
	//	for _, v := range autoSaveList {
	//		buf = buf + fmt.Sprintf("%v: %v\n", v.Autosave, v.Count)
	//	}
	//	buf = buf + "```"
	//}

	if count == 0 {
		buf = ""
	}
	buf = buf + "Syntax: `$vote-rewind <autosave number>`"
	return buf, count
}

func WriteRewindVotes() {
	var buf []byte
	buf, _ = json.Marshal(votes)
	err := ioutil.WriteFile(constants.VoteRewindFile, buf, 0644)
	if err != nil {
		botlog.DoLog(err.Error())
	}
}

func ReadRewindVotes() {
	buf, err := ioutil.ReadFile(constants.VoteRewindFile)
	if err != nil {
		botlog.DoLog(err.Error())
		return
	}
	newVotes := []rewindVoteData{}
	err = json.Unmarshal(buf, &newVotes)
	if err != nil {
		botlog.DoLog(err.Error())
		return
	}

	votes = newVotes
}
