package main

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/sclean"
	"ChatWire/support"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func handleDiscordMessages(s *discordgo.Session, m *discordgo.MessageCreate) {

	/* Ignore messages from self */
	if m.Author.ID == s.State.User.ID {
		return
	}

	message, _ := m.ContentWithMoreMentionsReplaced(s)
	if len(message) > 500 {
		message = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(message, 500))
	}
	message = sclean.UnicodeCleanup(message)

	/* Protect players from dumb mistakes with registration codes, even on other maps */
	/* Do this before we reject bot messages, to catch factorio chat on different maps/channels */
	if support.ProtectIdiots(message) {
		/* If they manage to post it into chat in Factorio on a different server,
		the message will be seen in discord but not factorio... eh whatever it still gets invalidated */
		buf := "You are supposed to type that into Factorio, not Discord... Invalidating code. Please read the directions more carefully..."
		_, err := s.ChannelMessageSend(m.ChannelID, buf)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}

	/* Throw away messages from bots */
	if m.Author.Bot {
		return
	}

	/* Command handling
	 * Factorio channel ONLY */
	if strings.EqualFold(cfg.Local.Channel.ChatChannel, m.ChannelID) && cfg.Local.Channel.ChatChannel != "" {

		/*
		 * Chat message handling
		 *  Don't bother if Factorio isn't running...
		 */
		if fact.FactorioBooted && fact.FactIsRunning {
			cwlog.DoLogCW("[" + m.Author.Username + "] " + message)

			/* Used for name matching */
			alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

			/* Remove control characters and discord markdown */
			cmess := sclean.RemoveDiscordMarkdown(message)

			/* Try to find factorio name, for registered players */
			dname := disc.GetFactorioNameFromDiscordID(m.Author.ID)
			nbuf := ""

			/* Name to lowercase */
			dnamelower := strings.ToLower(dname)
			fnamelower := strings.ToLower(m.Author.Username)

			/* Reduce names to letters only */
			dnamereduced := alphafilter.ReplaceAllString(dnamelower, "")
			fnamereduced := alphafilter.ReplaceAllString(fnamelower, "")

			/* Mark as seen, async */
			go func(factname string) {
				fact.UpdateSeen(factname)
			}(dname)

			/* Filter names... */
			corduser := sclean.UnicodeCleanup(m.Author.Username)
			cordnick := sclean.UnicodeCleanup(m.Member.Nick)
			factuser := sclean.UnicodeCleanup(dname)

			corduserlen := len(corduser)
			cordnicklen := len(cordnick)

			cordname := corduser

			/* On short names, try nickname... if not add number, if no name... discordID */
			if corduserlen < 5 {
				if cordnicklen >= 4 && cordnicklen < 18 {
					cordname = cordnick
				}
				cordnamelen := len(cordname)
				if cordnamelen > 0 {
					cordname = fnamereduced
				} else {
					cordname = fmt.Sprintf("ID#%s", m.Author.ID)
				}
			}

			/* Cap name length for safety/annoyance */
			cordname = sclean.TruncateString(cordname, 64)
			factuser = sclean.TruncateString(factuser, 64)

			/* Check if Discord name contains Factorio name, if not lets show both their names */
			if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {

				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] @%s (%s):[/color] %s", cordname, factuser, cmess)
			} else {
				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] %s:[/color] %s", cordname, cmess)
			}

			/* Send the final text to factorio */
			fact.FactChat(nbuf)

		}
	}
}
