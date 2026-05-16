package moderator

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

// MapExchange converts a Factorio map exchange string into the custom map
// generator files and immediately generates a new save from them.
func MapExchange(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if fact.FactorioBooted || fact.FactIsRunning {
		disc.InteractionEphemeralResponse(i, "Error:", "Factorio is currently running. You must stop Factorio first.")
		return
	}

	exchangeString := ""
	for _, arg := range i.ApplicationCommandData().Options {
		if arg.Type == discordgo.ApplicationCommandOptionString && strings.EqualFold(arg.Name, "exchange-string") {
			exchangeString = arg.StringValue()
			break
		}
	}

	if strings.TrimSpace(exchangeString) == "" {
		disc.InteractionEphemeralResponse(i, "Error:", "You must supply a map exchange string.")
		return
	}

	disc.InteractionEphemeralResponse(i, "Status:", "Converting map exchange string and generating a custom map.")

	fileName, err := fact.GenCustomMapFromExchange(exchangeString)
	if err != nil {
		msg := fmt.Sprintf("Unable to generate custom map: %v", err)
		disc.InteractionEphemeralResponseColor(i, "Error:", msg, glob.COLOR_RED)
		return
	}

	if i != nil && i.Member != nil && i.Member.User != nil {
		newHist := modupdate.ModHistoryItem{InfoItem: true,
			Name: "Generate Custom Map By: " + i.Member.User.Username, Notes: fileName, Date: time.Now()}
		modupdate.AddModHistory(newHist)
	}

	msg := fmt.Sprintf("Generated `%s` using `%s` map settings.", fileName, constants.CustomMapGeneratorName)
	disc.InteractionEphemeralResponse(i, "Map exchange", msg)
}
