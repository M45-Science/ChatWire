package moderator

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
)

const mapGeneratorSelectCustomID = "MapGenerator"

// MapGenerator shows a live dropdown of map generators discovered from the
// configured map generator folder and saves the selection for future map generation.
func MapGenerator(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	options := buildMapGeneratorOptions()
	if len(options) == 0 {
		disc.InteractionEphemeralResponse(i, "Error:", "No map generators are available.")
		return
	}

	embed := []*discordgo.MessageEmbed{{
		Title:       "Map generator",
		Description: "Map generator to use, selecting 'NONE' will use the map-preset setting instead. SELECT 'NONE' FOR MODS THAT REMOVE VANILLA RESOURCES.",
		Color:       glob.COLOR_WHITE,
	}}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    mapGeneratorSelectCustomID,
					Placeholder: "Choose map generator",
					Options:     options,
				},
			},
		},
	}

	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     embed,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	}

	if disc.DS == nil || i == nil || i.Interaction == nil {
		return
	}
	if err := disc.DS.InteractionRespond(i.Interaction, resp); err != nil {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to show map generator menu.")
	}
}

func buildMapGeneratorOptions() []discordgo.SelectMenuOption {
	names := getMapGenNames()
	if len(names) > constants.MaxMapResults {
		names = names[:constants.MaxMapResults]
	}

	options := make([]discordgo.SelectMenuOption, 0, len(names))
	for _, name := range names {
		isCurrent := strings.EqualFold(name, cfg.Local.Settings.MapGenerator) ||
			(strings.EqualFold(name, "none") && cfg.Local.Settings.MapGenerator == "")
		label := name
		if isCurrent {
			label += " (current)"
		}

		description := "Use a map generator from the configured folder."
		if strings.EqualFold(name, "none") {
			description = "Use the map preset setting instead."
		} else if strings.EqualFold(name, constants.CustomMapGeneratorName) {
			description = "Use the last custom map exchange settings."
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       name,
			Description: description,
			Default:     isCurrent,
		})
	}
	return options
}

func HandleMapGeneratorSelect(i *discordgo.InteractionCreate) {
	if i == nil {
		return
	}

	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		disc.InteractionEphemeralResponse(i, "Error:", "No map generator was selected.")
		return
	}

	selection := data.Values[0]
	displayName, err := applyMapGeneratorSelection(selection)
	if err != nil {
		disc.InteractionEphemeralResponseColor(i, "Error:", err.Error(), glob.COLOR_RED)
		return
	}

	if !cfg.WriteLCfg() {
		disc.InteractionEphemeralResponseColor(i, "Error:", "Unable to save cw-local config.", glob.COLOR_RED)
		return
	}

	msg := fmt.Sprintf("Map generator set to `%s`. It will be used the next time a map is generated.", displayName)
	disc.InteractionEphemeralResponse(i, "Status:", msg)
}

func applyMapGeneratorSelection(selection string) (string, error) {
	selection = strings.TrimSpace(selection)
	if !checkMapGen(selection) {
		return "", fmt.Errorf("map generator `%s` is no longer available", selection)
	}

	if shouldCacheMapGeneratorSelection(selection) {
		if _, _, err := cfg.CacheMapGenerator(selection); err != nil {
			return "", fmt.Errorf("map generator `%s` is available, but local fallback cache could not be updated: %w", selection, err)
		}
	}

	cfg.Local.Settings.MapGenerator = selection
	return selection, nil
}

func shouldCacheMapGeneratorSelection(selection string) bool {
	selection = strings.TrimSpace(selection)
	return selection != "" &&
		!strings.EqualFold(selection, "none") &&
		!strings.EqualFold(selection, constants.CustomMapGeneratorName)
}
