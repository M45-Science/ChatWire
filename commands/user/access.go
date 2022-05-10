package user

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

/* AccessServer locks PasswordListLock
 * This allows players to register, for discord roles and in-game perks */
func AccessServer(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsFactRunning() && 1 == 2 {
		respData := &discordgo.InteractionResponseData{Content: "Factorio isn't currently running."}
		resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
		s.InteractionRespond(i.Interaction, resp)
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
	if glob.PassList[i.Member.User.ID] != nil {
		delete(glob.PassList, i.Member.User.ID)
		cwlog.DoLogCW("Invalidating previous unused password...")
	}
	np := glob.PassData{
		Code:   password,
		DiscID: i.Member.User.ID,
		Time:   t.Unix(),
	}
	glob.PassList[i.Member.User.ID] = &np

	glob.PasswordListLock.Unlock()

	buf := fmt.Sprintf("1: Connect to the Factorio server: `%v-%v`\n", cfg.Local.Callsign, cfg.Local.Name)
	buf = buf + "2: Copy/Paste or type this command + code into the Factorio console/chat window:\n"
	buf = buf + fmt.Sprintf("`/register %v`\n", password)

	//Help
	buf = buf + fmt.Sprintf("\nTo find the server, you can search for `%v` in the server browser.\n", cfg.Global.GroupName)

	msg, isConfigured := fact.MakeSteamURL()
	if isConfigured {
		buf = buf + fmt.Sprintf("If you have Factorio on Steam, you can use this link to connect: %v\n", msg)
	}

	buf = buf + "You can use control-c and control-v to copy-paste the command and code (command on Mac)\n"
	buf = buf + "Make sure the line starts with forward-slash so you don't paste your code in chat.\n"
	buf = buf + "**DO NOT PASTE THE CODE IN CHAT OR DISCORD BY ACCIDENT! IF YOU DO RUN `/register` AGAIN TO INVALIDATE THE OLD CODE.**\n"
	buf = buf + "The code expires after 15 minutes, if you need another one just use `/register` again.\n"

	respData := &discordgo.InteractionResponseData{Content: buf, Flags: 1 << 6}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	s.InteractionRespond(i.Interaction, resp)
}
