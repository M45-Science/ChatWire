package support

import (
	"fmt"
	"strings"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/sclean"
)

func handleSoftModMsg(input *handleData) bool {
	/******************
	 * MSG AREA
	 ******************/
	if strings.HasPrefix(input.line, "[MSG]") {
		cwlog.DoLogGame(input.line)

		if input.wordListLen > 0 {
			ctext := strings.Join(input.wordList[1:], " ")

			/* Clean strings */
			cmess := sclean.UnicodeCleanup(ctext)
			cmess = sclean.EscapeDiscordMarkdown(cmess)
			cmess = sclean.RemoveFactorioTags(cmess)

			if len(cmess) > 500 {
				cmess = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(cmess, 500))
			}

			if strings.HasPrefix(cmess, "Research") {
				if cfg.Local.Options.HideResearch {
					return true
				}
			}

			fact.CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` **%s**", fact.CurrentGametime(), cmess))
		}

		return true
	}
	return false

}
