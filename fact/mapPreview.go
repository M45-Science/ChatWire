package fact

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"

	"github.com/bwmarrin/discordgo"
)

// MapPreviewURL preserves Factorio's compressed exchange data byte for byte.
// No preset lookup or seed inference is needed.
func MapPreviewURL(exchange string) (string, error) {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, exchange)
	if len(compact) > 512000 || !strings.HasPrefix(compact, ">>>") || !strings.HasSuffix(compact, "<<<") {
		return "", fmt.Errorf("invalid map exchange string")
	}
	encoded := strings.TrimSuffix(strings.TrimPrefix(compact, ">>>"), "<<<")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("invalid map exchange encoding")
	}
	return "https://mapgen.m45sci.xyz/#m=" + base64.RawURLEncoding.EncodeToString(data), nil
}

func mapPreviewMessage(url string) *discordgo.MessageSend {
	message := "New map preview: " + url
	if len(message) <= constants.MaxDiscordMsgLen {
		return &discordgo.MessageSend{Content: message}
	}
	// Never let the normal chat truncation turn a long exchange into a broken link.
	return &discordgo.MessageSend{
		Content: "New map preview: the full link is attached.",
		Files:   []*discordgo.File{{Name: "map-preview.txt", ContentType: "text/plain", Reader: strings.NewReader(url + "\n")}},
	}
}

func AnnounceMapPreview(exchange string) error {
	url, err := MapPreviewURL(exchange)
	if err != nil {
		return err
	}
	channel := cfg.Local.Channel.ChatChannel
	if channel == "" {
		return fmt.Errorf("map preview has no configured chat channel")
	}
	message := mapPreviewMessage(url)
	if len(message.Files) == 0 {
		LogCMS(channel, message.Content)
		return nil
	}
	if disc.DS == nil {
		return fmt.Errorf("Discord unavailable for map preview attachment")
	}
	_, err = disc.DS.ChannelMessageSendComplex(channel, message)
	return err
}
