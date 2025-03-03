package moderator

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	factUpdater.FetchLock.Lock()
	defer factUpdater.FetchLock.Unlock()

	disc.InteractionEphemeralResponse(i, "Status:", "Your file is uploading... please wait.")

	attachmentID := i.ApplicationCommandData().Options[0].Value.(string)
	attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL

	data, name, err := factUpdater.HttpGet(attachmentUrl)
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "File download failed.")
		return
	}

	err = os.WriteFile(name, data, 0555)

	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to write attached file.")
		return
	}

	sBuf := fmt.Sprintf("File: %v, Size: %v", name, humanize.Bytes(uint64(len(data))))
	fact.LogCMS(cfg.Local.Channel.ChatChannel, sBuf)
}
