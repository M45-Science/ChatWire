package user

import (
	"fmt"
	"sort"
	"strings"

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

var scoreboardUnits, scoreboardUnitsErr = durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")

func Scoreboard(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if scoreboardUnitsErr != nil {
		cwlog.DoLogCW("Scoreboard: failed to load duration units: %v", scoreboardUnitsErr)
		disc.InteractionEphemeralResponse(i, "Scoreboard", "An error occurred while generating the scoreboard. Please try again later.")
		return
	}

	//Make list of scores
	scores := make([]scoreData, 0)
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
	disc.InteractionEphemeralResponse(i, "Scoreboard:", formatScoreboard(scores))
}

func formatScoreboard(scores []scoreData) string {
	if len(scores) > 40 {
		scores = scores[:40]
	}

	var buf strings.Builder
	buf.WriteString("```")
	for i, p := range scores {

		n, _ := durafmt.ParseString(fmt.Sprintf("%vm", p.Score))
		timestr := n.LimitFirstN(2).Format(scoreboardUnits)
		fmt.Fprintf(&buf, "#%2v: %24v: %-15v\n", i+1, p.Name, timestr)
	}
	buf.WriteString("```")
	return buf.String()
}
