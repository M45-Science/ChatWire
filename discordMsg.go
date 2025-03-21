package main

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"
	"ChatWire/util"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Protect against spam
func handleDiscordMessages(s *discordgo.Session, m *discordgo.MessageCreate) {

	message, _ := m.ContentWithMoreMentionsReplaced(s)
	message = sclean.UnicodeCleanup(message)

	messageHandlerLock.Lock()
	defer messageHandlerLock.Unlock()

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

	/* Ignore messages from self */
	if m.Author.ID == s.State.User.ID {
		return
	}

	/* Throw away messages from bots */
	if m.Author.Bot {
		return
	}

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

	//Limit size
	if len(message) > 500 {
		message = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(message, 500))
	}

	//Kill continuity.
	glob.BootMessage = nil
	glob.UpdateMessage = nil

	/* Command handling
	 * Factorio channel ONLY */
	if strings.EqualFold(cfg.Local.Channel.ChatChannel, m.ChannelID) && cfg.Local.Channel.ChatChannel != "" {

		/* Used for name matching */
		alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

		/* Remove control characters and discord markdown */
		cleanedMessage := sclean.RemoveDiscordMarkdown(message)

		/* Try to find factorio name, for registered players */
		factName := disc.GetFactorioNameFromDiscordID(m.Author.ID)

		/* Name to lowercase */
		factNameLower := strings.ToLower(factName)
		discNameLower := strings.ToLower(m.Author.GlobalName)

		/* Reduce names to letters only */
		factNameReduced := alphafilter.ReplaceAllString(factNameLower, "")
		discNameReduced := alphafilter.ReplaceAllString(discNameLower, "")

		/* Mark as seen, async */
		go func(factname string) {
			fact.UpdateSeen(factname)
		}(factName)

		/* Filter names... */
		discordName := sclean.UnicodeCleanup(m.Author.GlobalName)
		factorioName := sclean.UnicodeCleanup(factName)

		/* Just in case of weird nickname */
		discordNameLen := len(discordName)
		if discordNameLen < 2 {
			discordName = m.Author.Username
		}
		/* Just in case of weird username */
		discordNameLen = len(discordName)
		if discordNameLen < 2 {
			discordName = fmt.Sprintf("ID#%v", m.Member.User.ID)
		}

		/* Cap name length for safety/annoyance */
		discordName = sclean.TruncateString(discordName, 64)
		factorioName = sclean.TruncateString(factorioName, 64)

		namePrefix := ""
		interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Member: m.Member}}
		if disc.CheckAdmin(interaction) {
			namePrefix = "(Admin) "
		} else if disc.CheckModerator(interaction) {
			namePrefix = "(Moderator) "
		} else if util.IsPatreon(m.Author.ID) || disc.CheckSupporter(interaction) {
			namePrefix = "(Supporter) "
		} else if disc.CheckVeteran(interaction) {
			namePrefix = "(Veteran) "
		} else if disc.CheckRegular(interaction) {
			namePrefix = "(Regular) "
		} else if disc.CheckMember(interaction) {
			namePrefix = "(Member) "
		}

		/* Check if Discord name contains Factorio name, if not lets show both their names */
		output := ""
		if factName != "" && !strings.Contains(factNameReduced, discNameReduced) && !strings.Contains(discNameReduced, factNameReduced) {
			output = fmt.Sprintf("[Discord] %v%v(%v): %v", namePrefix, discordName, factorioName, cleanedMessage)
		} else {
			output = fmt.Sprintf("[Discord] %v%v: %v", namePrefix, discordName, cleanedMessage)
		}

		/* Send the final text to factorio */
		if fact.FactorioBooted && fact.FactIsRunning {
			fact.FactChat(output)
		}
		cwlog.DoLogGame(output)
	}
}
