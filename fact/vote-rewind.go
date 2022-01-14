package fact

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/glob"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func ResetTotalVotes() {
	for vpos, _ := range glob.VoteBox.Votes {
		glob.VoteBox.Votes[vpos].TotalVotes = 0
	}
	TallyRewindVotes()
	WriteRewindVotes()
}

//Don't use if already locked
func VoidAllVotes() {
	for vpos, _ := range glob.VoteBox.Votes {
		glob.VoteBox.Votes[vpos].Voided = true
	}
	TallyRewindVotes()
	WriteRewindVotes()
}

func PrintVote(v glob.RewindVoteData) string {
	buf := fmt.Sprintf("%v: autosave #%v (%v ago)", v.Name, v.AutosaveNum, time.Since(v.Time).Round(time.Second).String())
	return buf
}

//EXPECTS LOCKED VOTEBOX
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

			//buf = buf + printVote(v)
			//buf = buf + " (expired)\n"
			//visVotes++
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

	//Clear if nothing generated
	if visVotes == 0 {
		buf = ""
	}
	buf = buf + fmt.Sprintf("`Total valid votes: %v -- (need %v for the same autosave)`\n", validVotes, constants.VotesNeededRewind)

	//Reset tally, recount
	glob.VoteBox.Tally = []glob.VoteTallyData{}
	for _, v := range glob.VoteBox.Votes {
		for apos, a := range glob.VoteBox.Tally {
			if v.AutosaveNum == a.Autosave {
				//Same autosave, tally
				glob.VoteBox.Tally[apos] = glob.VoteTallyData{Autosave: a.Autosave, Count: a.Count + 1}
				continue
			}
		}
		//Different autosave, add to list
		glob.VoteBox.Tally = append(glob.VoteBox.Tally, glob.VoteTallyData{Autosave: v.AutosaveNum, Count: 1})
	}

	buf = buf + "Syntax: `$vote-rewind <autosave number>`"
	return buf, validVotes
}

//Expects locked votebox
func WriteRewindVotes() bool {
	finalPath := constants.VoteRewindFile
	tempPath := constants.VoteRewindFile + "." + cfg.Local.ServerCallsign + ".tmp"

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	glob.VoteBox.Version = "0.0.1"

	if err := enc.Encode(glob.VoteBox); err != nil {
		botlog.DoLog("WriteRewindVotes: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("WriteRewindVotes: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteRewindVotes: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		botlog.DoLog("Couldn't rename VoteRewindFile file.")
		return false
	}

	return true
}

func ReadRewindVotes() bool {
	_, err := os.Stat(constants.VoteRewindFile)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadRewindVotes: os.Stat failed")
		return true
	} else { //Just read the config

		file, err := ioutil.ReadFile(constants.VoteRewindFile)

		if file != nil && err == nil {
			temp := CreateVoteContainer()

			err := json.Unmarshal([]byte(file), &temp)
			if err != nil {
				botlog.DoLog("ReadRewindVotes: Unmarshal failure")
				botlog.DoLog(err.Error())
				return false
			}

			glob.VoteBox = temp
			botlog.DoLog("ReadRewindVotes: Successfully read file")
			return true
		} else {
			botlog.DoLog("ReadRewindVotes: ReadFile failure")
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
