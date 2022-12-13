package user

import (
	"fmt"
	"sort"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"

	"ChatWire/disc"
	"ChatWire/glob"
)

type scoreData struct {
	Name  string
	Score int64
}

/**************************
 * Show useful info about a server and it's settings
 *************************/
func Scoreboard(s *discordgo.Session, i *discordgo.InteractionCreate) {

	units, err := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")
	if err != nil {
		panic(err)
	}

	//Make list of scores
	buf := "```"
	scores := []scoreData{}
	glob.PlayerListLock.RLock()
	for _, p := range glob.PlayerList {
		if p.Minutes != 0 {
			scores = append(scores, scoreData{Name: p.Name, Score: p.Minutes})
		}
	}
	glob.PlayerListLock.RUnlock()

	//Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	//Print scoreboard
	count := 0
	numScores := len(scores) - 1
	for x := numScores; x >= 0; x-- {
		p := scores[x]
		if count >= 40 {
			break
		}

		n, _ := durafmt.ParseString(fmt.Sprintf("%vm", p.Score))
		timestr := n.Format(units)
		buf = buf + fmt.Sprintf("#%2v: %24v: %-15v\n", count+1, p.Name, timestr)

		count++
	}
	buf = buf + "```"
	disc.EphemeralResponse(s, i, "Scoreboard: (top 40)", buf)
}
