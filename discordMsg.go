package main

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Protect against spam
func handleDiscordMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	messageHandlerLock.Lock()
	defer messageHandlerLock.Unlock()

	/* Ignore messages from self */
	if m.Author.ID == s.State.User.ID {
		return
	}

	message, _ := m.ContentWithMoreMentionsReplaced(s)
	message = sclean.UnicodeCleanup(message)

	//Show attachments
	if len(m.Attachments) > 0 {
		message = message + "Attachments: "
	}
	urls := ""
	for _, item := range m.Attachments {
		if item.URL != "" {
			if urls != "" {
				urls = urls + ", "
			}
			urls = urls + item.ContentType + ": " + item.URL
		}
	}
	message = message + urls

	//Show stickers
	stickers := ""
	if len(m.StickerItems) > 0 {
		message = message + "Stickers: "
	}
	for _, item := range m.StickerItems {
		if item.Name != "" {
			if stickers != "" {
				stickers = stickers + ", "
			}
			stickers = stickers + item.Name
		}
	}
	message = message + stickers

	if message == "" {
		return
	}

m.

	//Limit size
	if len(message) > 500 {
		message = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(message, 500))
	}

	//Kill continuity.
	glob.BootMessage = nil
	glob.UpdateMessage = nil

	/* Protect players from dumb mistakes with registration codes, even on other maps */
	/* Do this before we reject bot messages, to catch factorio chat on different maps/channels */
	if support.ProtectIdiots(message) {
		replyChan := m.ChannelID

		/* If they manage to post it into chat in Factorio on a different server,
		the message will be seen in discord but not factorio... eh whatever it still gets invalidated */
		buf := "You are supposed to type that into Factorio, not Discord... Invalidating code. Please read the directions more carefully..."
		embed := &discordgo.MessageEmbed{Title: "WARNING!!!", Description: m.Author.Username + ": " + buf, Color: glob.COLOR_RED}
		disc.SmartWriteDiscordEmbed(replyChan, embed)

		/* Delete message if possible */
		err := s.ChannelMessageDelete(replyChan, m.ID)
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
			cwlog.DoLogGame("[Discord] " + m.Author.Username + ": " + message)

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
