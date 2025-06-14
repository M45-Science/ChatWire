package moderator

import (
	"ChatWire/cfg"
	"github.com/bwmarrin/discordgo"
	"os"
	"testing"
)

func TestConfigHours(t *testing.T) {
	oldwd, _ := os.Getwd()
	dir := t.TempDir()
	os.Chdir(dir)
	defer os.Chdir(oldwd)

	origEnable := cfg.Local.Options.PlayHourEnable
	origStart := cfg.Local.Options.PlayStartHour
	origEnd := cfg.Local.Options.PlayEndHour
	defer func() {
		cfg.Local.Options.PlayHourEnable = origEnable
		cfg.Local.Options.PlayStartHour = origStart
		cfg.Local.Options.PlayEndHour = origEnd
	}()

	ic := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:   discordgo.InteractionApplicationCommand,
		Member: &discordgo.Member{User: &discordgo.User{Username: "tester"}},
		Data: discordgo.ApplicationCommandInteractionData{
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "enabled", Type: discordgo.ApplicationCommandOptionBoolean, Value: true},
				{Name: "start-hour", Type: discordgo.ApplicationCommandOptionInteger, Value: float64(3)},
				{Name: "end-hour", Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)},
			},
		},
	}}

	ConfigHours(nil, ic)

	if !cfg.Local.Options.PlayHourEnable || cfg.Local.Options.PlayStartHour != 3 || cfg.Local.Options.PlayEndHour != 5 {
		t.Fatalf("options not set correctly")
	}
}
