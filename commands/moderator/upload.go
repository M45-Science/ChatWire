package moderator

import (
	"ChatWire/disc"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
)

func UploadFile(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	attachmentID := i.ApplicationCommandData().Options[0].Value.(string)
	attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL
	disc.InteractionEphemeralResponse(i, "Status:", "Your file is uploading... please wait.")

	data, name, err := factUpdater.HttpGet(attachmentUrl)
	if err != nil {
		//
	}
	sBuf := fmt.Sprintf("File: %v, Size: %v", name, humanize.Bytes(uint64(len(data))))
	disc.DS.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Upload complete: " + sBuf,
	})

}
