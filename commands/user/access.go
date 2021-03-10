package user

import (
	"fmt"
	"time"

	"../../cfg"
	"../../constants"
	"../../disc"
	"../../fact"
	"../../glob"
	"github.com/bwmarrin/discordgo"
	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

//AccessServer locks PasswordListLock
func AccessServer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	//Do before lock
	g := xkcdpwgen.NewGenerator()
	g.SetNumWords(3)
	g.SetCapitalize(false)
	g.SetDelimiter("-")
	password := g.GeneratePasswordString()

	t := time.Now()

	glob.PasswordListLock.Lock()
	for i := 0; i <= glob.PasswordMax && i <= constants.MaxPasswords; i++ {
		if glob.PasswordID[i] == m.Author.ID {
			glob.PasswordList[i] = ""
			glob.PasswordID[i] = ""
			glob.PasswordTime[i] = 0

			fact.CMS(cfg.Local.ChannelData.LogID, "Invalidating previous unused password...")

		}
	}
	if glob.PasswordMax >= constants.MaxPasswords {
		glob.PasswordMax = 0
	}
	glob.PasswordList[glob.PasswordMax] = password
	glob.PasswordID[glob.PasswordMax] = m.Author.ID
	glob.PasswordTime[glob.PasswordMax] = t.Unix()
	glob.PasswordMax++
	glob.PasswordListLock.Unlock()

	servername := cfg.Local.ServerCallsign + "-" + cfg.Local.Name

	dmChannel := disc.SmartChannelCreate(m.Author.ID)
	disc.SmartWriteDiscord(dmChannel.ID, fmt.Sprintf("**How to register:**\n\nOn the Factorio server '%s', copy-paste or type this in the console/chat in Factorio.\n***(PASTE OR TYPE IN FACTORIO, NOT IN DISCORD)***\n`/register %s`\n\n**If unused, the code expires after five minutes**, you can use `$register` again to get another.\nThe code will only work once, and is specific to your Discord ID, so **DO NOT share/post the code.**\nYour Discord name can only be registered to one Factorio name, and visa-versa.\n\n**If you accidentally paste the code publicly:**\nUse `$register` again, to get a new code (this destroys old code)\nPlease also delete your message, if possible.\n\n**Factorio Chat/Console:**\nThe key is typically ~ or \\` (tilde or tick). If it isn't, you can see what the key is (or change it) in settings.\nThe setting is called 'Toggle chat (and lua console)' under 'Controls settings'.\n", servername, password))
	fact.CMS(m.ChannelID, "Access code was direct-messaged to you!\n\nPaste or type the *command and code* into Factorio.\n**DO NOT PASTE THE CODE ON DISCORD.**")
}
