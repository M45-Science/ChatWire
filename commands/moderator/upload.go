package moderator

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	disc.InteractionEphemeralResponse(i, "Status", "Processing, please wait...")
	factUpdater.FetchLock.Lock()
	defer factUpdater.FetchLock.Unlock()

	var found bool
	var msg *discordgo.Message
	for _, item := range i.ApplicationCommandData().Options {
		tName := item.Name

		switch tName {
		case "save-game":
		case "mod-list":
		case "mod-settings":
			//
		default:
			continue
		}

		found = true
		msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+tName+" file is uploading.", glob.COLOR_GREEN)

		attachmentID := item.Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

		data, name, err := factUpdater.HttpGet(attachmentUrl)
		if err != nil {
			msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+tName+" file failed while downloading.", glob.COLOR_RED)
			continue
		}

		err = os.WriteFile(name, data, 0555)

		if err != nil {
			msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", "Your "+tName+" file failed while writing.", glob.COLOR_RED)
			continue
		}

		sBuf := fmt.Sprintf("Downloaded %v: %v, Size: %v", tName, name, humanize.Bytes(uint64(len(data))))
		msg = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, msg, "Status", sBuf, glob.COLOR_GREEN)

	}
	if !found {
		disc.InteractionEphemeralResponse(i, "Error:", "You must supply a file to upload.")
	}
}
