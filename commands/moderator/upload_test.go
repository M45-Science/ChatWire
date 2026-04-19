package moderator

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestAttachmentURL(t *testing.T) {
	t.Run("valid attachment", func(t *testing.T) {
		i := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Resolved: &discordgo.ApplicationCommandInteractionDataResolved{
						Attachments: map[string]*discordgo.MessageAttachment{
							"a1": {URL: "https://example.invalid/file.zip"},
						},
					},
				},
			},
		}
		item := &discordgo.ApplicationCommandInteractionDataOption{Value: "a1"}

		url, ok := attachmentURL(i, item)
		if !ok {
			t.Fatal("expected attachment lookup to succeed")
		}
		if url != "https://example.invalid/file.zip" {
			t.Fatalf("unexpected url: %q", url)
		}
	})

	t.Run("missing attachment id", func(t *testing.T) {
		i := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Resolved: &discordgo.ApplicationCommandInteractionDataResolved{
						Attachments: map[string]*discordgo.MessageAttachment{},
					},
				},
			},
		}
		item := &discordgo.ApplicationCommandInteractionDataOption{Value: "missing"}

		if _, ok := attachmentURL(i, item); ok {
			t.Fatal("expected missing attachment lookup to fail")
		}
	})

	t.Run("non string value", func(t *testing.T) {
		i := &discordgo.InteractionCreate{}
		item := &discordgo.ApplicationCommandInteractionDataOption{Value: 123}

		if _, ok := attachmentURL(i, item); ok {
			t.Fatal("expected non-string attachment value to fail")
		}
	})
}
