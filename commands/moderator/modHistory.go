package moderator

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func ModHistoryCmd(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	//
}

func ListHistory(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	buf := ""
	for i, item := range modupdate.ModHistory {
		buf = buf + fmt.Sprintf("ID#%03v: Name: %v\nVersion: %10vnDate: %v\n",
			i, item.Name, item.Version, item.Date)
	}
	if buf == "" {
		buf = "History is empty."
	}
	disc.InteractionEphemeralResponse(i, "Mod History", buf)
}

func ClearHistory(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	modupdate.ModHistory = []modupdate.ModHistoryData{}
	modupdate.WriteModHistory()
	disc.InteractionEphemeralResponse(i, "Mod History", "Mod history was cleared.")

}

func BlacklistItem(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	//
}
