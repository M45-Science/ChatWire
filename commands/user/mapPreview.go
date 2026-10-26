package user

import (
	"encoding/base64"
	"fmt"
	"strings"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

func MapPreview(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if !fact.FactorioBooted || !fact.FactIsRunning {
		disc.InteractionEphemeralResponse(i, "Map preview", "Factorio isn't running.")
		return
	}
	// Acknowledge before the bounded RCON request so Discord does not time out.
	disc.InteractionEphemeralResponse(i, "Map preview", "Reading the current map...")
	reply, err := fact.ExecuteRCON("map-preview")
	if err == nil {
		reply, err = mapPreviewURLFromReply(reply)
	}
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Map preview", err.Error())
		return
	}
	if len(reply) > 4000 {
		disc.InteractionEphemeralFileResponse(i, "Map preview", "The full link is attached.", "map-preview.txt", []byte(reply+"\n"))
		return
	}
	disc.InteractionEphemeralResponse(i, "Map preview", reply)
}

func mapPreviewURLFromReply(reply string) (string, error) {
	const prefix = "https://mapgen.m45sci.xyz/#m="
	url := strings.TrimSpace(reply)
	if strings.HasPrefix(url, prefix) {
		data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(url, prefix))
		if err == nil && len(data) > 0 {
			return url, nil
		}
	}
	return "", fmt.Errorf("Map preview unavailable. The running map needs the updated SoftMod.")
}
