package user

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* AccessServer locks PasswordListLock
 * This allows players to register, for discord roles and in-game perks */
func Register(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if !fact.IsFactRunning() {
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
		buf = buf + "**NOTICE: Invalidating previous unused registration code.**\n"
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
	buf = buf + "The code expires after 5 minutes, if you need another one just use `/register` again.\n"
	if cfg.Local.Options.Whitelist {
		buf = buf + "**NOTICE: This is a MEMBERS-ONLY server, if you haven't reached the MEMBER level in-game, you will be unable to connect. If this is the case, use the /register command on a PUBLIC server.**\n"
	}

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "How to complete registration:", Description: buf})

	//1 << 6 is ephemeral/private, don't use disc.EphemeralResponse (logged)
	respData := &discordgo.InteractionResponseData{Embeds: elist, Flags: 1 << 6}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	err := s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		return
	}
}
