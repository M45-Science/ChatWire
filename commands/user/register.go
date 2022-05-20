package user

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* AccessServer locks PasswordListLock
 * This allows players to register, for discord roles and in-game perks */
func Register(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsFactRunning() && 1 == 2 {
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: "Factorio isn't currently running."}
		disc.InteractionResponse(s, i, embed)
		return
	}
	/* Do before lock */
	g := xkcdpwgen.NewGenerator()
	g.SetNumWords(3)
	g.SetCapitalize(false)
	g.SetDelimiter("-")
	password := g.GeneratePasswordString()

	t := time.Now()

	glob.PasswordListLock.Lock()
	didDelete := false
	if glob.PassList[i.Member.User.ID] != nil {
		delete(glob.PassList, i.Member.User.ID)
		didDelete = true
	}
	np := glob.PassData{
		Code:   password,
		DiscID: i.Member.User.ID,
		Time:   t.Unix(),
	}
	glob.PassList[i.Member.User.ID] = &np

	glob.PasswordListLock.Unlock()

	buf := ""
	if didDelete {
		buf = buf + "**NOTICE: Invalidating previous registration code.**\n"
	}
	buf = buf + fmt.Sprintf("1: Connect to the Factorio server: `%v-%v`\n", cfg.Local.Callsign, cfg.Local.Name)
	buf = buf + "2: Copy/Paste or type this registration command into the Factorio console/chat window:\n"
	buf = buf + fmt.Sprintf("`/register %v`\n", password)

	//Help
	buf = buf + fmt.Sprintf("\nTo find the server, you can search for `%v` in the server browser.\n", cfg.Global.GroupName)

	msg, isConfigured := fact.MakeSteamURL()
	if isConfigured {
		buf = buf + fmt.Sprintf("If you have Factorio on Steam, you can use this link to connect: %v\n", msg)
	}

	buf = buf + "You can use control-c and control-v to copy-paste the command and code (command on Mac)\n"
	buf = buf + "The code expires after 15 minutes, if you need another one just use `/register` again.\n"
	buf = buf + "**IF YOU ACCIDENTLY SHARE THE CODE, RUN `/REGISTER` AGAIN TO INVALIDATE THE CODE.**\n"

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "How to complete registration:", Description: buf})

	respData := &discordgo.InteractionResponseData{Embeds: elist, Flags: 1 << 6}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	err := s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
}
