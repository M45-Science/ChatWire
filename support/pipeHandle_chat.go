package support

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	embed "github.com/Clinet/discordgo-embed"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

var slurList []string = []string{
	"nigger",
	"faggot",
	"kike",
	"kyke",
}

func slurFilter(name string, input *handleData) bool {

	for _, word := range input.lowerWordList {
		for _, slur := range slurList {
			if strings.HasPrefix(word, slur) {
				fact.WriteBanBy(name, "Use of extreme slurs", "[Auto-Ban]")
				return true
			} else if strings.HasSuffix(word, slur) {
				fact.WriteBanBy(name, "Use of extreme slurs", "[Auto-Ban]")
				return true
			}
		}
	}

	return false
}

func handleChatMsg(input *handleData) bool {
	/************************
	 * FACTORIO CHAT MESSAGES
	 ************************/
	if strings.HasPrefix(input.noDatestamp, "[CHAT]") || strings.HasPrefix(input.noDatestamp, "[SHOUT]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			input.noDatestampList[1] = strings.ReplaceAll(input.noDatestampList[1], ":", "")
			pname := input.noDatestampList[1]

			if pname != "<server>" {

				var nores int = glob.NoResponseCount

				if !cfg.Global.Options.DisableSpamProtect {
					glob.ChatterLock.Lock()
					now := time.Now()

					//Do not ban for chat spam if game is lagging
					if nores < 5 && fact.PlayerLevelGet(pname, true) != 255 {

						//Lower score if they cool off
						if now.Sub(glob.ChatterList[pname]) > constants.SpamResetThres {
							glob.ChatterSpamScore[pname] = 0
							glob.ChatterList[pname] = now
						} else if now.Sub(glob.ChatterList[pname]) > constants.SpamCoolThres {
							if glob.ChatterSpamScore[pname] > 0 {
								glob.ChatterSpamScore[pname]--
							}
							glob.ChatterList[pname] = now
						}

						//Normal chat, add one point
						if now.Sub(glob.ChatterList[pname]) < constants.SpamSlowThres {
							glob.ChatterSpamScore[pname]++
							glob.ChatterList[pname] = now
							//Super spammy, add two points
						} else if now.Sub(glob.ChatterList[pname]) < constants.SpamFastThres {
							glob.ChatterSpamScore[pname] += 2
							glob.ChatterList[pname] = now
						}

						if glob.ChatterSpamScore[pname] > constants.SpamScoreWarning {
							fact.FactWhisper(pname, "*** SPAMMING / FLOODING WARNING! ***")
							cwlog.DoLogCW("Spam warning: %v: Score: %v", pname, glob.ChatterSpamScore[pname])
						}
						if glob.ChatterSpamScore[pname] > constants.SpamScoreLimit {
							fact.WriteBanBy(pname, "Spamming/Flooding", "[Auto-Ban]")
							cwlog.DoLogCW("Spam BANNED: %v: Score: %v", pname, glob.ChatterSpamScore[pname])

							glob.PlayerListLock.Lock()
							if glob.PlayerList[pname] != nil &&
								!glob.PlayerList[pname].AlreadyBanned {
								glob.PlayerList[pname].AlreadyBanned = true
							}
							glob.PlayerListLock.Unlock()

							glob.ChatterSpamScore[pname] = 0
						}

						if slurFilter(pname, input) {
							return true
						}
					} else {
						/* Lower score if server isn't responding */
						if glob.ChatterSpamScore[pname] > 0 {
							glob.ChatterSpamScore[pname]--
						}
					}

					glob.ChatterLock.Unlock()
				}

				cmess := strings.Join(input.noDatestampList[2:], " ")
				cmess = sclean.UnicodeCleanup(cmess)
				cmess = sclean.EscapeDiscordMarkdown(cmess)
				cmess = sclean.RemoveFactorioTags(cmess)

				if len(cmess) > 500 {
					cmess = fmt.Sprintf("%s**(message cut, too long!)**", sclean.TruncateStringEllipsis(cmess, 500))
				}

				if cmess == "" {
					return true
				}

				/* Yeah, on different thread please. */
				go func(ptemp string) {
					fact.UpdateSeen(ptemp)
				}(pname)

				did := disc.GetDiscordIDFromFactorioName(pname)
				dname := disc.GetNameFromID(did)
				avatar := disc.GetDiscordAvatarFromId(did, 64)
				factname := sclean.UnicodeCleanup(pname)
				factname = sclean.TruncateString(factname, 25)

				fbuf := ""
				/* Filter Factorio names */

				factname = sclean.UnicodeCleanup(factname)
				factname = sclean.EscapeDiscordMarkdown(factname)
				if dname != "" {
					fbuf = fmt.Sprintf("`%v` **%s**: %s", fact.Gametime, factname, cmess)
				} else {
					fbuf = fmt.Sprintf("`%v` %s: %s", fact.Gametime, factname, cmess)
				}

				/* Remove all but letters */
				filter, _ := regexp.Compile("[^a-zA-Z]+")

				/* Name to lowercase */
				dnamelower := strings.ToLower(dname)
				fnamelower := strings.ToLower(pname)

				/* Reduce to letters only */
				dnamereduced := filter.ReplaceAllString(dnamelower, "")
				fnamereduced := filter.ReplaceAllString(fnamelower, "")

				/* If we find Discord name, and Discord name and Factorio name don't contain the same name */
				if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {
					/* Slap data into embed format. */
					myembed := embed.NewEmbed().
						SetAuthor("@"+dname, avatar).
						SetDescription(fbuf).
						MessageEmbed

						/* Send it off! */
					disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, myembed)
					return true
				}
				fact.CMS(cfg.Local.Channel.ChatChannel, fbuf)
				return true
			}
		}
	}
	return false
}
