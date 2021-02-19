package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"./cfg"
	"./commands"
	"./disc"
	"./fact"
	"./support"
	"github.com/bwmarrin/discordgo"
)

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	input, _ := m.ContentWithMoreMentionsReplaced(s)
	ctext := support.StripControlAndSubSpecial(input)
	log.Print("[" + m.Author.Username + "] " + ctext)

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	//Command stuff
	//AUX channel
	if m.ChannelID == cfg.Local.ChannelData.LogID {
		if strings.HasPrefix(ctext, cfg.Global.DiscordCommandPrefix) {
			empty := []string{}

			slen := len(ctext)

			if slen > 1 {

				args := strings.Split(ctext, " ")
				arglen := len(args)

				if arglen > 0 {
					name := strings.ToLower(args[0])
					if arglen > 1 {
						commands.RunCommand(name[1:], s, m, args[1:arglen])
					} else {
						commands.RunCommand(name[1:], s, m, empty)
					}
				}
			}
			return
		}
	} else if m.ChannelID == cfg.Local.ChannelData.ChatID { //Factorio channel
		if strings.HasPrefix(ctext, cfg.Global.DiscordCommandPrefix) {
			empty := []string{}

			slen := len(ctext)

			if slen > 1 {

				args := strings.Split(ctext, " ")
				arglen := len(args)

				if arglen > 0 {
					name := strings.ToLower(args[0])
					if arglen > 1 {
						commands.RunCommand(name[1:], s, m, args[1:arglen])
					} else {
						commands.RunCommand(name[1:], s, m, empty)
					}
				}
			}
			return
		}
	}

	//Block everything but chat
	if m.ChannelID != cfg.Local.ChannelData.ChatID {
		return
	}

	//Chat message handling
	if fact.IsFactorioBooted() { // Don't bother if we arne't running...
		if !strings.HasPrefix(ctext, "!") { //block mee6 commands

			alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

			//Clean strings
			cmess := support.StripControlAndSubSpecial(ctext)
			//cmess = unidecode.Unidecode(cmess)

			//Remove factorio tags
			rega := regexp.MustCompile(`\[/[^][]+\]`) //remove close tags [/color]

			regc := regexp.MustCompile(`\[color=(.*?)\]`) //remove [color=*]
			regd := regexp.MustCompile(`\[font=(.*?)\]`)  //remove [font=*]

			regf := regexp.MustCompile(`\*+`) //Remove discord markdown
			regg := regexp.MustCompile(`\~+`)
			regh := regexp.MustCompile(`\_+`)

			for regc.MatchString(cmess) || regd.MatchString(cmess) {
				//Remove colors/fonts
				cmess = regc.ReplaceAllString(cmess, "")
				cmess = regd.ReplaceAllString(cmess, "")
			}
			for rega.MatchString(cmess) {
				//Filter close tags
				cmess = rega.ReplaceAllString(cmess, "")
			}
			for regf.MatchString(cmess) || regg.MatchString(cmess) || regh.MatchString(cmess) {
				//Filter discord tags
				cmess = regf.ReplaceAllString(cmess, "")
				cmess = regg.ReplaceAllString(cmess, "")
				cmess = regh.ReplaceAllString(cmess, "")
			}

			if len(cmess) > 500 {
				cmess = fmt.Sprintf("%s...(cut, too long!)", support.TruncateString(cmess, 500))
			}

			if cmess == "" {
				return
			}

			dname := disc.GetFactorioNameFromDiscordID(m.Author.ID)
			nbuf := ""

			//Name to lowercase
			dnamelower := strings.ToLower(dname)
			fnamelower := strings.ToLower(m.Author.Username)

			//Reduce names to letters only
			dnamereduced := alphafilter.ReplaceAllString(dnamelower, "")
			fnamereduced := alphafilter.ReplaceAllString(fnamelower, "")

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(dname)

			//Filter names...
			corduser := support.StripControlAndSubSpecial(m.Author.Username)
			cordnick := support.StripControlAndSubSpecial(m.Member.Nick)
			factuser := support.StripControlAndSubSpecial(dname)

			corduserlen := len(corduser)
			cordnicklen := len(cordnick)

			cordname := corduser

			//On short names, try nickname... if not add number, if no name... discordID
			if corduserlen < 5 {
				if cordnicklen >= 4 && cordnicklen < 18 {
					cordname = cordnick
				}
				cordnamelen := len(cordname)
				if cordnamelen > 0 {
					cordname = fmt.Sprintf("%s#%s", fnamereduced, m.Author.Discriminator)
				} else {
					cordname = fmt.Sprintf("ID#%s", m.Author.ID)
				}
			}

			//Cap name length
			cordname = support.TruncateString(cordname, 25)
			factuser = support.TruncateString(factuser, 25)

			//If we find discord name, and discord name... and factorio name don't contain the same name
			if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {

				nbuf = fmt.Sprintf("/cchat [color=0,1,1][DISCORD][/color] [color=1,1,0]@%s[/color] [color=0,0.5,0](%s):[/color] %s%s[/color]", cordname, factuser, fact.RandomColor(false), cmess)
			} else {
				nbuf = fmt.Sprintf("/cchat [color=0,1,1][DISCORD][/color] [color=1,1,0]@%s:[/color] %s%s[/color]", cordname, fact.RandomColor(false), cmess)
			}

			fact.WriteFact(nbuf)
		}
	}
}
