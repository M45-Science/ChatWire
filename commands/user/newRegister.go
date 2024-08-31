package user

import (
	"fmt"
	"strings"
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
func Register(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	if !fact.FactIsRunning {
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: "Factorio isn't currently running."}
		disc.InteractionResponse(i, embed)
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
	buf = buf + fmt.Sprintf("1: Open **Factorio** and connect to: `%v-%v`\n", strings.ToUpper(cfg.Local.Callsign), cfg.Local.Name)
	buf = buf + "2: Copy/Paste or type this registration command and code **into Factorio:**\n"
	buf = buf + fmt.Sprintf("`/register %v`\n", password)

	//Help
	buf = buf + fmt.Sprintf("\nTo find the server, you can search for `%v` in the **Factorio server browser.**\n", cfg.Global.GroupName)

	msg, isConfigured := fact.MakeSteamURL()
	if isConfigured {
		buf = buf + fmt.Sprintf("If you have Factorio on Steam, you can use this link to connect: %v\n", msg)
	}

	buf = buf + "\nThe reason this is necessary:\nYour Discord and Factorio names can be different, so this is the only way to find your player-level in Factorio.\n"
	if cfg.Local.Options.RegularsOnly {
		buf = buf + "**NOTICE: This is a REGULARS-ONLY server, if you haven't reached the REGULAR level in-game, you will be unable to connect. If this is the case, use the /register command on a PUBLIC server.**\n"
	} else if cfg.Local.Options.MembersOnly {
		buf = buf + "**NOTICE: This is a MEMBERS-ONLY server, if you haven't reached the MEMBER level in-game, you will be unable to connect. If this is the case, use the /register command on a PUBLIC server.**\n"
	}

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "How to complete registration:", Description: buf})

	//1 << 6 is ephemeral/private, don't use disc.EphemeralResponse (logged)
	respData := &discordgo.InteractionResponseData{Embeds: elist, Flags: 1 << 6}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	err := disc.DS.InteractionRespond(i.Interaction, resp)
	if err != nil {
		return
	}
}
