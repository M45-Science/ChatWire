package user

import (
	"fmt"
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"

	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

type scoreData struct {
	Name  string
	Score int64
}

func Scoreboard(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	units, err := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")
	if err != nil {
		cwlog.DoLogCW("Scoreboard: failed to load duration units: %v", err)
		disc.InteractionEphemeralResponse(i, "Scoreboard", "An error occurred while generating the scoreboard. Please try again later.")
		return
	}

	//Make list of scores
	buf := "```"
	scores := []scoreData{}
	glob.PlayerListLock.RLock()
	for _, p := range glob.PlayerList {
		if p.Level >= 2 {
			scores = append(scores, scoreData{Name: p.Name, Score: p.Minutes})
		}
	}
	glob.PlayerListLock.RUnlock()

	//Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	//Print scoreboard
	count := 0
	numScores := len(scores) - 1
	if numScores > 40 {
		numScores = 40
	}
	for x := 0; x < numScores; x++ {
		p := scores[x]

		n, _ := durafmt.ParseString(fmt.Sprintf("%vm", p.Score))
		timestr := n.LimitFirstN(2).Format(units)
		buf = buf + fmt.Sprintf("#%2v: %24v: %-15v\n", count+1, p.Name, timestr)

		count++
	}
	buf = buf + "```"
	disc.InteractionEphemeralResponse(i, "Scoreboard:", buf)
}
